import { type Route } from '@playwright/test';

// Mock data for analytics API
const mockSessionMetrics = {
  total_sessions: 156,
  unique_users: 24,
  avg_duration_seconds: 2340,
  total_duration_seconds: 365040,
  date_range: {
    start: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
    end: new Date().toISOString(),
  },
  by_hour: Array.from({ length: 24 }, (_, i) => ({
    hour: i,
    sessions: Math.floor(Math.random() * 20) + 1,
    unique_users: Math.floor(Math.random() * 10) + 1,
  })),
  by_day: Array.from({ length: 7 }, (_, i) => ({
    date: new Date(Date.now() - (6 - i) * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
    sessions: Math.floor(Math.random() * 30) + 10,
    unique_users: Math.floor(Math.random() * 15) + 5,
  })),
};

const mockDashboardTrends = {
  sessions: {
    current: 156,
    previous: 142,
    change_percent: 10,
  },
  users: {
    current: 24,
    previous: 22,
    change_percent: 9,
  },
  duration: {
    current: 2340,
    previous: 2100,
    change_percent: 11,
  },
};

const mockRealtimeStats = {
  active_sessions: 8,
  active_users: 6,
  sessions_last_hour: 12,
  avg_active_duration: 1800,
};

const mockCommandRiskSummary = {
  total_commands: 1234,
  high_risk_commands: 45,
  medium_risk_commands: 123,
  low_risk_commands: 1066,
  unique_users: 24,
  unique_targets: 38,
};

const mockAnomalySummary = {
  total: 5,
  by_severity: {
    critical: 2,
    high: 2,
    medium: 1,
    low: 0,
  },
  by_type: {
    unusual_access_time: 1,
    privileged_escalation: 1,
    unusual_location: 1,
    bulk_data_access: 1,
    command_injection: 1,
  },
  by_status: {
    open: 3,
    investigating: 1,
    resolved: 1,
    false_positive: 0,
  },
  resolved_this_period: 1,
  avg_resolution_time_hours: 4.5,
};

const mockUserActivity = [
  {
    id: 'user-1',
    user_id: 'user-1',
    user_name: 'John Doe',
    user_email: 'john.doe@example.com',
    session_count: 15,
    total_duration_seconds: 32400,
    avg_duration_seconds: 2160,
    unique_targets: 5,
    last_activity: new Date().toISOString(),
    risk_score: 15,
  },
  {
    id: 'user-2',
    user_id: 'user-2',
    user_name: 'Jane Smith',
    user_email: 'jane.smith@example.com',
    session_count: 12,
    total_duration_seconds: 28800,
    avg_duration_seconds: 2400,
    unique_targets: 4,
    last_activity: new Date(Date.now() - 3600000).toISOString(),
    risk_score: 8,
  },
  {
    id: 'user-3',
    user_id: 'user-3',
    user_name: 'Bob Johnson',
    user_email: 'bob.johnson@example.com',
    session_count: 8,
    total_duration_seconds: 14400,
    avg_duration_seconds: 1800,
    unique_targets: 3,
    last_activity: new Date(Date.now() - 7200000).toISOString(),
    risk_score: 22,
  },
];

const mockCommandFrequency = [
  {
    id: 'cmd-1',
    command: 'sudo',
    base_command: 'sudo',
    frequency: 234,
    unique_users: 12,
    risk_level: 'low',
    last_used: new Date().toISOString(),
  },
  {
    id: 'cmd-2',
    command: 'rm -rf',
    base_command: 'rm',
    frequency: 23,
    unique_users: 5,
    risk_level: 'high',
    last_used: new Date(Date.now() - 3600000).toISOString(),
  },
  {
    id: 'cmd-3',
    command: 'cat /etc/passwd',
    base_command: 'cat',
    frequency: 12,
    unique_users: 3,
    risk_level: 'medium',
    last_used: new Date(Date.now() - 7200000).toISOString(),
  },
  {
    id: 'cmd-4',
    command: 'dd if=/dev/zero',
    base_command: 'dd',
    frequency: 2,
    unique_users: 1,
    risk_level: 'critical',
    last_used: new Date(Date.now() - 86400000).toISOString(),
  },
];

const mockTopUsers = [
  {
    user_id: 'user-1',
    user_name: 'John Doe',
    user_email: 'john.doe@example.com',
    session_count: 25,
    total_duration_seconds: 54000,
  },
  {
    user_id: 'user-2',
    user_name: 'Jane Smith',
    user_email: 'jane.smith@example.com',
    session_count: 20,
    total_duration_seconds: 48000,
  },
];

const mockTopTargets = [
  {
    target_id: 'target-1',
    target_name: 'prod-db-01',
    target_type: 'database',
    session_count: 45,
    unique_users: 8,
  },
  {
    target_id: 'target-2',
    target_name: 'app-server-02',
    target_type: 'server',
    session_count: 32,
    unique_users: 6,
  },
];

const mockComplianceDashboard = {
  overall_score: 87,
  control_count: 50,
  compliant_count: 44,
  non_compliant_count: 3,
  skipped_count: 3,
  last_assessed: new Date().toISOString(),
  frameworks: [
    {
      framework: 'SOC2',
      score: 92,
      controls: 35,
      compliant: 32,
    },
    {
      framework: 'ISO27001',
      score: 82,
      controls: 15,
      compliant: 12,
    },
  ],
  recent_assessments: [
    {
      id: 'assessment-1',
      framework: 'SOC2',
      status: 'completed',
      score: 92,
      created_at: new Date(Date.now() - 86400000).toISOString(),
    },
  ],
};

const mockComplianceControls = [
  {
    id: 'ctrl-1',
    control_id: 'AC-001',
    name: 'Access Control',
    framework: 'SOC2',
    category: 'Access',
    status: 'compliant',
    description: 'System access controls are properly implemented',
    last_assessed: new Date().toISOString(),
    evidence_count: 5,
  },
  {
    id: 'ctrl-2',
    control_id: 'AC-002',
    name: 'Authentication',
    framework: 'SOC2',
    category: 'Access',
    status: 'non_compliant',
    description: 'Multi-factor authentication not enforced for all users',
    last_assessed: new Date().toISOString(),
    evidence_count: 2,
  },
  {
    id: 'ctrl-3',
    control_id: 'AC-003',
    name: 'Authorization',
    framework: 'SOC2',
    category: 'Access',
    status: 'compliant',
    description: 'Role-based access control implemented',
    last_assessed: new Date().toISOString(),
    evidence_count: 8,
  },
];

const mockAnomalies = [
  {
    id: 'anom-1',
    type: 'unusual_access_time',
    severity: 'critical',
    title: 'Unusual After-Hours Access',
    description: 'User accessed production system at 3 AM',
    user_id: 'user-1',
    user_name: 'Admin User',
    target_id: 'target-1',
    target_name: 'prod-server-01',
    session_id: 'sess-1',
    confidence_score: 0.92,
    risk_score: 95,
    detected_at: new Date(Date.now() - 3600000).toISOString(),
    status: 'open',
    indicators: [],
    assigned_to: '',
    resolution_notes: '',
  },
  {
    id: 'anom-2',
    type: 'privileged_escalation',
    severity: 'high',
    title: 'Sudden Privilege Escalation',
    description: 'User escalated privileges multiple times in short period',
    user_id: 'user-2',
    user_name: 'Regular User',
    target_id: 'target-1',
    target_name: 'prod-server-01',
    session_id: 'sess-2',
    confidence_score: 0.85,
    risk_score: 75,
    detected_at: new Date(Date.now() - 7200000).toISOString(),
    status: 'investigating',
    indicators: [],
    assigned_to: '',
    resolution_notes: '',
  },
  {
    id: 'anom-3',
    type: 'unusual_location',
    severity: 'medium',
    title: 'Access from Unusual Location',
    description: 'Login detected from previously unseen geographic location',
    user_id: 'user-3',
    user_name: 'Dev User',
    target_id: 'target-2',
    target_name: 'staging-db-01',
    session_id: 'sess-3',
    confidence_score: 0.75,
    risk_score: 50,
    detected_at: new Date(Date.now() - 10800000).toISOString(),
    status: 'open',
    indicators: [],
    assigned_to: '',
    resolution_notes: '',
  },
  {
    id: 'anom-4',
    type: 'bulk_data_access',
    severity: 'high',
    title: 'Bulk Data Export Detected',
    description: 'Large volume of data exported from database',
    user_id: 'user-1',
    user_name: 'Admin User',
    target_id: 'target-3',
    target_name: 'prod-db-01',
    session_id: 'sess-4',
    confidence_score: 0.88,
    risk_score: 85,
    detected_at: new Date(Date.now() - 14400000).toISOString(),
    status: 'open',
    indicators: [],
    assigned_to: '',
    resolution_notes: '',
  },
  {
    id: 'anom-5',
    type: 'command_injection',
    severity: 'critical',
    title: 'Potential Command Injection',
    description: 'Suspicious command pattern detected in session',
    user_id: 'user-4',
    user_name: 'Contractor',
    target_id: 'target-1',
    target_name: 'prod-server-01',
    session_id: 'sess-5',
    confidence_score: 0.95,
    risk_score: 98,
    detected_at: new Date(Date.now() - 18000000).toISOString(),
    status: 'resolved',
    resolved_at: new Date(Date.now() - 3600000).toISOString(),
    resolution_notes: 'Investigated and confirmed as false positive - legitimate automation script',
    indicators: [],
    assigned_to: '',
  },
];

const mockTimeSeries = Array.from({ length: 24 }, (_, i) => ({
  timestamp: new Date(Date.now() - (23 - i) * 3600000).toISOString(),
  value: Math.floor(Math.random() * 20) + 5,
  label: `${i}:00`,
}));

// Analytics API handler
export const mockAnalyticsRoutes = async (route: Route) => {
  const url = route.request().url();
  const method = route.request().method();

  // GET /analytics/sessions/metrics
  if (url.includes('/analytics/sessions/metrics') && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockSessionMetrics),
    });
    return;
  }

  // GET /analytics/dashboard/trends
  if (url.includes('/analytics/dashboard/trends') && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockDashboardTrends),
    });
    return;
  }

  // GET /analytics/dashboard
  if (url.includes('/analytics/dashboard') && method === 'GET' && !url.includes('/trends')) {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        metrics: mockSessionMetrics,
        trends: mockDashboardTrends,
        realtime: mockRealtimeStats,
      }),
    });
    return;
  }

  // GET /analytics/timeseries
  if (url.includes('/analytics/timeseries') && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockTimeSeries),
    });
    return;
  }

  // GET /analytics/users/activity
  if (url.includes('/analytics/users/activity') && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: mockUserActivity,
        pagination: { total: mockUserActivity.length, offset: 0, limit: 20 },
      }),
    });
    return;
  }

  // GET /analytics/users/:id/activity
  if (url.match(/\/analytics\/users\/[^\/]+\/activity/) && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockUserActivity[0]),
    });
    return;
  }

  // GET /analytics/users/top
  if (url.includes('/analytics/users/top') && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockTopUsers),
    });
    return;
  }

  // GET /analytics/targets/top
  if (url.includes('/analytics/targets/top') && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockTopTargets),
    });
    return;
  }

  // GET /analytics/commands/frequency
  if (url.includes('/analytics/commands/frequency') && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: mockCommandFrequency,
        pagination: { total: mockCommandFrequency.length, offset: 0, limit: 20 },
      }),
    });
    return;
  }

  // GET /analytics/commands/risk-summary
  if (url.includes('/analytics/commands/risk-summary') && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockCommandRiskSummary),
    });
    return;
  }

  // POST /analytics/commands/export
  if (url.includes('/analytics/commands/export') && method === 'POST') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        download_url: '/downloads/commands-export.csv',
        expires_at: new Date(Date.now() + 3600000).toISOString(),
      }),
    });
    return;
  }

  // POST /analytics/users/export
  if (url.includes('/analytics/users/export') && method === 'POST') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        download_url: '/downloads/users-export.csv',
        expires_at: new Date(Date.now() + 3600000).toISOString(),
      }),
    });
    return;
  }

  // GET /analytics/realtime
  if (url.includes('/analytics/realtime') && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockRealtimeStats),
    });
    return;
  }

  // GET /analytics/anomalies
  if (url.includes('/analytics/anomalies') && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: mockAnomalies,
        pagination: { total: mockAnomalies.length, offset: 0, limit: 20 },
      }),
    });
    return;
  }

  // PATCH /analytics/anomalies/:id
  if (url.match(/\/analytics\/anomalies\/[^\/]+$/) && method === 'PATCH') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ id: url.split('/').pop(), status: 'investigating' }),
    });
    return;
  }

  // GET /analytics/compliance/summary
  if (url.includes('/analytics/compliance/summary') && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        overall_score: mockComplianceDashboard.overall_score,
        control_count: mockComplianceDashboard.control_count,
        compliant_count: mockComplianceDashboard.compliant_count,
        non_compliant_count: mockComplianceDashboard.non_compliant_count,
        last_assessed: mockComplianceDashboard.last_assessed,
      }),
    });
    return;
  }

  // Default fallback for analytics routes
  await route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({}),
  });
};

// Compliance API handler
export const mockComplianceRoutes = async (route: Route) => {
  const url = route.request().url();
  const method = route.request().method();

  // GET /compliance/dashboard
  if (url.includes('/compliance/dashboard') && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockComplianceDashboard),
    });
    return;
  }

  // GET /compliance/controls
  if (url.includes('/compliance/controls') && method === 'GET' && !url.includes('/compliance/controls/')) {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: mockComplianceControls,
        pagination: { total: mockComplianceControls.length, offset: 0, limit: 20 },
      }),
    });
    return;
  }

  // GET /compliance/controls/:id
  if (url.match(/\/compliance\/controls\/[^\/]+$/) && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockComplianceControls[0]),
    });
    return;
  }

  // POST /compliance/assessments/run
  if (url.includes('/compliance/assessments/run') && method === 'POST') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        assessment_id: 'assessment-new',
        status: 'running',
        started_at: new Date().toISOString(),
      }),
    });
    return;
  }

  // GET /compliance/assessments/:id/status
  if (url.match(/\/compliance\/assessments\/[^\/]+\/status/) && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'assessment-1',
        status: 'completed',
        progress: 100,
        started_at: new Date(Date.now() - 300000).toISOString(),
        completed_at: new Date().toISOString(),
      }),
    });
    return;
  }

  // GET /compliance/exceptions
  if (url.includes('/compliance/exceptions') && method === 'GET' && !url.includes('/compliance/exceptions/')) {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: [],
        pagination: { total: 0, offset: 0, limit: 20 },
      }),
    });
    return;
  }

  // POST /compliance/exceptions
  if (url.includes('/compliance/exceptions') && method === 'POST' && !url.includes('/revoke')) {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'exception-new',
        control_id: 'AC-001',
        status: 'approved',
        created_at: new Date().toISOString(),
      }),
    });
    return;
  }

  // GET /compliance/frameworks
  if (url.includes('/compliance/frameworks') && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { framework: 'SOC2', name: 'SOC 2', description: 'Service Organization Control 2', control_count: 35 },
        { framework: 'ISO27001', name: 'ISO 27001', description: 'Information Security Management', control_count: 15 },
        { framework: 'HIPAA', name: 'HIPAA', description: 'Health Insurance Portability and Accountability Act', control_count: 42 },
        { framework: 'PCI-DSS', name: 'PCI DSS', description: 'Payment Card Industry Data Security Standard', control_count: 28 },
      ]),
    });
    return;
  }

  // GET /compliance/anomalies
  if (url.includes('/compliance/anomalies') && method === 'GET' && !url.includes('/compliance/anomalies/')) {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: mockAnomalies,
        pagination: { total: mockAnomalies.length, offset: 0, limit: 20 },
      }),
    });
    return;
  }

  // GET /compliance/anomalies/:id
  if (url.match(/\/compliance\/anomalies\/[^\/]+$/) && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockAnomalies[0]),
    });
    return;
  }

  // PATCH /compliance/anomalies/:id
  if (url.match(/\/compliance\/anomalies\/[^\/]+$/) && method === 'PATCH') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ ...mockAnomalies[0], status: 'investigating' }),
    });
    return;
  }

  // POST /compliance/anomalies/:id/acknowledge
  if (url.includes('/acknowledge') && method === 'POST') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ ...mockAnomalies[0], status: 'investigating' }),
    });
    return;
  }

  // GET /compliance/anomalies/summary
  if (url.includes('/compliance/anomalies/summary') && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockAnomalySummary),
    });
    return;
  }

  // POST /compliance/anomalies/bulk-update
  if (url.includes('/compliance/anomalies/bulk-update') && method === 'POST') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ updated: 2, failed: [] }),
    });
    return;
  }

  // POST /compliance/anomalies/export
  if (url.includes('/compliance/anomalies/export') && method === 'POST') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        download_url: '/exports/anomalies.csv',
        expires_at: new Date(Date.now() + 3600000).toISOString(),
      }),
    });
    return;
  }

  // GET /compliance/anomalies/related
  if (url.includes('/compliance/anomalies/related') && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: mockAnomalies.slice(0, 2),
        pagination: { total: 2, offset: 0, limit: 20, has_more: false },
      }),
    });
    return;
  }

  // GET /compliance/ransomware/indicators
  if (url.includes('/compliance/ransomware/indicators') && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { indicator_type: 'file_encryption', description: 'Bulk file encryption detected', severity: 'critical', detected_count: 0 },
        { indicator_type: 'suspicious_commands', description: 'High-risk command patterns', severity: 'high', detected_count: 2 },
        { indicator_type: 'unusual_access', description: 'Access outside normal patterns', severity: 'medium', detected_count: 5 },
      ]),
    });
    return;
  }

  // POST /compliance/emergency/lockdown
  if (url.includes('/compliance/emergency/lockdown') && method === 'POST' && !url.includes('/release')) {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        lockdown_id: 'lockdown-1',
        status: 'active',
        initiated_at: new Date().toISOString(),
      }),
    });
    return;
  }

  // GET /compliance/emergency/lockdown/:id
  if (url.match(/\/compliance\/emergency\/lockdown\/[^\/]+$/) && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'lockdown-1',
        status: 'active',
        reason: 'Security incident',
        initiated_at: new Date().toISOString(),
        initiated_by: 'admin',
      }),
    });
    return;
  }

  // POST /compliance/emergency/lockdown/:id/release
  if (url.includes('/release') && method === 'POST') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'released',
        released_at: new Date().toISOString(),
      }),
    });
    return;
  }

  // POST /compliance/reports/generate
  if (url.includes('/compliance/reports/generate') && method === 'POST') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        report_id: 'report-1',
        status: 'generating',
      }),
    });
    return;
  }

  // GET /compliance/reports/:id
  if (url.match(/\/compliance\/reports\/[^\/]+$/) && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'report-1',
        framework: 'SOC2',
        status: 'completed',
        download_url: '/downloads/report-1.pdf',
        created_at: new Date().toISOString(),
      }),
    });
    return;
  }

  // Default fallback for compliance routes
  await route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({}),
  });
};
