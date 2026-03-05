export const validateEmail = (email: string): boolean => {
  const re = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return re.test(email);
};

// SECURITY: Password validation must match backend requirements (12+ chars, complexity)
export const validatePassword = (password: string): { valid: boolean; errors: string[] } => {
  const errors: string[] = [];

  if (password.length < 12) {
    errors.push('Password must be at least 12 characters long');
  }
  if (password.length > 128) {
    errors.push('Password must not exceed 128 characters');
  }
  if (!/[a-z]/.test(password)) {
    errors.push('Password must contain at least one lowercase letter');
  }
  if (!/[A-Z]/.test(password)) {
    errors.push('Password must contain at least one uppercase letter');
  }
  if (!/\d/.test(password)) {
    errors.push('Password must contain at least one number');
  }
  if (!/[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?`~]/.test(password)) {
    errors.push('Password must contain at least one special character');
  }

  // Check for common weak patterns
  const lower = password.toLowerCase();
  const weakPatterns = ['password', 'admin', 'welcome', 'qwerty', '123456'];
  for (const pattern of weakPatterns) {
    if (lower.includes(pattern)) {
      errors.push('Password contains a commonly used pattern');
      break;
    }
  }

  return {
    valid: errors.length === 0,
    errors,
  };
};

export const validateURL = (url: string): boolean => {
  try {
    new URL(url);
    return true;
  } catch {
    return false;
  }
};

export const validateIP = (ip: string): boolean => {
  const ipv4Regex = /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;
  const ipv6Regex = /^(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$/;
  return ipv4Regex.test(ip) || ipv6Regex.test(ip);
};

export const validatePort = (port: number | string): boolean => {
  const p = typeof port === 'string' ? parseInt(port, 10) : port;
  return !isNaN(p) && p > 0 && p <= 65535;
};

// SECURITY: Comprehensive HTML entity encoding to prevent XSS
// Escapes all characters that could be used for HTML injection
export const sanitizeInput = (input: string): string => {
  const map: Record<string, string> = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#x27;',
    '`': '&#x60;',
    '/': '&#x2F;',
  };
  return input.replace(/[&<>"'`/]/g, (char) => map[char] || char);
};

// SECURITY: Validate that a URL is same-origin (relative path only)
export const isSafeRedirectUrl = (url: string): boolean => {
  return url.startsWith('/') && !url.startsWith('//') && !url.includes('://');
};
