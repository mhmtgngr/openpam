-- Anomaly Deduplication Migration
-- Adds correlation-based deduplication to prevent duplicate anomalies
-- from being created for the same underlying event or pattern.

-- Add correlation_id and correlation metadata columns to anomaly_detections
ALTER TABLE anomaly_detections
ADD COLUMN IF NOT EXISTS correlation_id UUID,
ADD COLUMN IF NOT EXISTS correlation_key VARCHAR(255),
ADD COLUMN IF NOT EXISTS duplicate_count INTEGER NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS is_duplicate BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN IF NOT EXISTS first_detection_id UUID,
ADD COLUMN IF NOT EXISTS merged_into_id UUID;

-- Create unique index on correlation_key to prevent duplicates at database level
-- NULL values are allowed (not all anomalies have a correlation key)
CREATE UNIQUE INDEX IF NOT EXISTS idx_anomaly_correlation_key
    ON anomaly_detections(correlation_key)
    WHERE correlation_key IS NOT NULL AND is_duplicate = false;

-- Create index for correlation lookups
CREATE INDEX IF NOT EXISTS idx_anomaly_correlation_id
    ON anomaly_detections(correlation_id)
    WHERE correlation_id IS NOT NULL;

-- Create index for finding duplicates
CREATE INDEX IF NOT EXISTS idx_anomaly_duplicates
    ON anomaly_detections(tenant_id, anomaly_type, user_id, is_duplicate)
    WHERE is_duplicate = false;

-- Create index for first detection references
CREATE INDEX IF NOT EXISTS idx_anomaly_first_detection
    ON anomaly_detections(first_detection_id)
    WHERE first_detection_id IS NOT NULL;

-- Create index for merged detection references
CREATE INDEX IF NOT EXISTS idx_anomaly_merged_into
    ON anomaly_detections(merged_into_id)
    WHERE merged_into_id IS NOT NULL;

-- Add function to generate correlation key from anomaly properties
CREATE OR REPLACE FUNCTION generate_anomaly_correlation_key(
    p_tenant_id UUID,
    p_anomaly_type VARCHAR,
    p_user_id UUID,
    p_target_host VARCHAR,
    p_indicators JSONB
) RETURNS VARCHAR AS $$
DECLARE
    key_parts TEXT[];
    indicator_hash TEXT;
BEGIN
    -- Build correlation key from stable properties
    key_parts := ARRAY[
        p_tenant_id::TEXT,
        p_anomaly_type,
        COALESCE(p_user_id::TEXT, 'null'),
        COALESCE(p_target_host, 'null')
    ];

    -- Add stable indicator hash for pattern-based anomalies
    IF p_indicators IS NOT NULL THEN
        -- Extract stable indicators for correlation (e.g., command pattern, IP range)
        SELECT COALESCE(
            p_indicators->>'command_pattern',
            p_indicators->>'ip_range',
            p_indicators->>'file_pattern',
            md5(p_indicators::TEXT)
        ) INTO indicator_hash;

        IF indicator_hash IS NOT NULL THEN
            key_parts := array_append(key_parts, indicator_hash);
        END IF;
    END IF;

    -- Generate deterministic key
    RETURN digest(array_to_string(key_parts, '|'), 'sha256');
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Add trigger function to auto-generate correlation key on insert
CREATE OR REPLACE FUNCTION set_anomaly_correlation_key()
RETURNS TRIGGER AS $$
BEGIN
    -- Only generate if not explicitly set
    IF NEW.correlation_key IS NULL THEN
        NEW.correlation_key := generate_anomaly_correlation_key(
            NEW.tenant_id,
            NEW.anomaly_type,
            NEW.user_id,
            NEW.target_host,
            NEW.indicators
        );
    END IF;

    -- Set correlation_id to match ID for first detection
    IF NEW.correlation_id IS NULL THEN
        NEW.correlation_id := NEW.id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Add trigger to auto-generate correlation key
DROP TRIGGER IF EXISTS trigger_set_anomaly_correlation_key ON anomaly_detections;
CREATE TRIGGER trigger_set_anomaly_correlation_key
    BEFORE INSERT ON anomaly_detections
    FOR EACH ROW
    EXECUTE FUNCTION set_anomaly_correlation_key();

-- Add updated_at trigger for new columns
DROP TRIGGER IF EXISTS update_anomaly_detections_updated_at ON anomaly_detections;
CREATE TRIGGER update_anomaly_detections_updated_at
    BEFORE UPDATE ON anomaly_detections
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create view for active anomalies (excludes duplicates and ignored)
CREATE OR REPLACE VIEW active_anomalies AS
SELECT
    id,
    tenant_id,
    anomaly_type,
    user_id,
    session_id,
    target_host,
    severity,
    confidence_score,
    risk_score,
    title,
    description,
    indicators,
    detection_method,
    detected_at,
    model_version,
    status,
    assigned_to,
    resolution_notes,
    resolved_at,
    resolved_by,
    auto_triggered,
    auto_action_taken,
    correlation_id,
    correlation_key,
    duplicate_count,
    created_at,
    updated_at
FROM anomaly_detections
WHERE is_duplicate = false
  AND status != 'ignored';

-- Create view for anomaly statistics
CREATE OR REPLACE VIEW anomaly_statistics AS
SELECT
    tenant_id,
    anomaly_type,
    severity,
    status,
    COUNT(*) FILTER (WHERE is_duplicate = false) as count,
    COUNT(*) FILTER (WHERE is_duplicate = false AND status = 'open') as open_count,
    COUNT(*) FILTER (WHERE is_duplicate = false AND status = 'investigating') as investigating_count,
    COUNT(*) FILTER (WHERE is_duplicate = false AND status = 'resolved') as resolved_count,
    AVG(risk_score) FILTER (WHERE is_duplicate = false) as avg_risk_score,
    MAX(risk_score) FILTER (WHERE is_duplicate = false) as max_risk_score,
    COUNT(DISTINCT correlation_id) FILTER (WHERE is_duplicate = false) as unique_correlations,
    SUM(duplicate_count) as total_duplicates
FROM anomaly_detections
GROUP BY tenant_id, anomaly_type, severity, status;

-- Create function to mark duplicates when inserting new anomaly
CREATE OR REPLACE FUNCTION mark_anomaly_duplicates()
RETURNS TRIGGER AS $$
DECLARE
    existing_id UUID;
    duplicate_count INTEGER;
BEGIN
    -- Check for existing anomaly with same correlation key
    SELECT id, duplicate_count
    INTO existing_id, duplicate_count
    FROM anomaly_detections
    WHERE correlation_key = NEW.correlation_key
      AND id != NEW.id
      AND is_duplicate = false
      AND status NOT IN ('resolved', 'false_positive', 'ignored')
    LIMIT 1;

    -- If existing found, mark new as duplicate
    IF FOUND THEN
        NEW.is_duplicate := true;
        NEW.merged_into_id := existing_id;
        NEW.first_detection_id := existing_id;

        -- Update existing anomaly's duplicate count
        UPDATE anomaly_detections
        SET duplicate_count = duplicate_count + 1,
            updated_at = NOW()
        WHERE id = existing_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Add trigger to mark duplicates on insert
DROP TRIGGER IF EXISTS trigger_mark_anomaly_duplicates ON anomaly_detections;
CREATE TRIGGER trigger_mark_anomaly_duplicates
    AFTER INSERT ON anomaly_detections
    FOR EACH ROW
    EXECUTE FUNCTION mark_anomaly_duplicates();

-- Create function to merge duplicate anomalies
CREATE OR REPLACE FUNCTION merge_duplicate_anomalies(p_anomaly_id UUID)
RETURNS TABLE(merged_count INTEGER) AS $$
DECLARE
    correlation_val VARCHAR(255);
BEGIN
    -- Get correlation key of target anomaly
    SELECT correlation_key INTO correlation_val
    FROM anomaly_detections
    WHERE id = p_anomaly_id
    LIMIT 1;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- Mark all other anomalies with same correlation as duplicates
    UPDATE anomaly_detections
    SET is_duplicate = true,
        merged_into_id = p_anomaly_id,
        updated_at = NOW()
    WHERE correlation_key = correlation_val
      AND id != p_anomaly_id
      AND is_duplicate = false;

    -- Update duplicate count on target
    UPDATE anomaly_detections
    SET duplicate_count = (
        SELECT COUNT(*)
        FROM anomaly_detections
        WHERE correlation_key = correlation_val
          AND id != p_anomaly_id
    ),
    updated_at = NOW()
    WHERE id = p_anomaly_id;

    RETURN QUERY SELECT COUNT(*) FROM anomaly_detections WHERE merged_into_id = p_anomaly_id;
END;
$$ LANGUAGE plpgsql;

-- Add comments for documentation
COMMENT ON COLUMN anomaly_detections.correlation_id IS 'Unique identifier for a correlation group of related anomalies';
COMMENT ON COLUMN anomaly_detections.correlation_key IS 'Deterministic key generated from anomaly properties for deduplication';
COMMENT ON COLUMN anomaly_detections.duplicate_count IS 'Number of duplicate anomalies merged into this one';
COMMENT ON COLUMN anomaly_detections.is_duplicate IS 'True if this anomaly was marked as a duplicate of another';
COMMENT ON COLUMN anomaly_detections.first_detection_id IS 'ID of the first detection in this correlation group';
COMMENT ON COLUMN anomaly_detections.merged_into_id IS 'ID of the primary anomaly this duplicate was merged into';
COMMENT ON FUNCTION generate_anomaly_correlation_key IS 'Generates a deterministic correlation key from anomaly properties';
COMMENT ON FUNCTION mark_anomaly_duplicates IS 'Trigger function that marks new anomalies as duplicates if correlation key exists';
COMMENT ON FUNCTION merge_duplicate_anomalies IS 'Merges all anomalies with the same correlation key into the specified anomaly';
COMMENT ON VIEW active_anomalies IS 'View of non-duplicate, non-ignored anomalies for queries';
COMMENT ON VIEW anomaly_statistics IS 'Aggregated statistics about anomalies grouped by type, severity, and status';
