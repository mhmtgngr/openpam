import React, { useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { Shield, Eye, EyeOff } from 'lucide-react';
import { useAuth } from '@/contexts/AuthContext';
import { Input } from '@/components/common';
import { Button } from '@/components/common';

interface LoginFormData {
  email: string;
  password: string;
  mfaCode: string;
}

export const LoginForm: React.FC = () => {
  const { login, mfaRequired } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  const [formData, setFormData] = useState<LoginFormData>({
    email: '',
    password: '',
    mfaCode: '',
  });
  const [showPassword, setShowPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [errors, setErrors] = useState<Partial<Record<keyof LoginFormData, string>>>({});

  const validate = (): boolean => {
    const newErrors: Partial<Record<keyof LoginFormData, string>> = {};

    if (!formData.email) {
      newErrors.email = 'Email is required';
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = 'Invalid email format';
    }

    if (!formData.password) {
      newErrors.password = 'Password is required';
    }

    if (mfaRequired && !formData.mfaCode) {
      newErrors.mfaCode = 'MFA code is required';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validate()) return;

    setIsLoading(true);
    try {
      await login(formData.email, formData.password, formData.mfaCode);
      // Navigation is handled by the auth context
    } catch {
      // Error is handled by the axios interceptor
    } finally {
      setIsLoading(false);
    }
  };

  const handleChange = (field: keyof LoginFormData) => (value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    if (errors[field]) {
      setErrors((prev) => ({ ...prev, [field]: undefined }));
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-900 px-4">
      <div className="w-full max-w-md">
        {/* Logo and Header */}
        <div className="mb-8 text-center">
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-primary-600/20">
            <Shield className="h-8 w-8 text-primary-500" />
          </div>
          <h1 className="text-3xl font-bold text-white">OpenPAM</h1>
          <p className="mt-2 text-gray-400">
            {mfaRequired ? 'Enter your MFA code' : 'Sign in to your account'}
          </p>
        </div>

        {/* Login Form */}
        <div className="card">
          <form onSubmit={handleSubmit} className="space-y-6">
            <Input
              label="Email"
              type="email"
              placeholder="you@example.com"
              value={formData.email}
              onChange={(e) => handleChange('email')(e.target.value)}
              error={errors.email}
              disabled={mfaRequired}
              autoComplete="email"
              autoFocus={!mfaRequired}
            />

            {!mfaRequired && (
              <Input
                label="Password"
                type={showPassword ? 'text' : 'password'}
                placeholder="••••••••"
                value={formData.password}
                onChange={(e) => handleChange('password')(e.target.value)}
                error={errors.password}
                rightIcon={
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="focus:outline-none"
                  >
                    {showPassword ? (
                      <EyeOff className="h-4 w-4" />
                    ) : (
                      <Eye className="h-4 w-4" />
                    )}
                  </button>
                }
                autoComplete="current-password"
              />
            )}

            {mfaRequired && (
              <Input
                label="MFA Code"
                type="text"
                placeholder="123456"
                value={formData.mfaCode}
                onChange={(e) => handleChange('mfaCode')(e.target.value)}
                error={errors.mfaCode}
                maxLength={6}
                pattern="[0-9]*"
                inputMode="numeric"
                autoFocus
              />
            )}

            <Button
              type="submit"
              className="w-full"
              isLoading={isLoading}
            >
              {mfaRequired ? 'Verify' : 'Sign In'}
            </Button>

            {mfaRequired && (
              <button
                type="button"
                onClick={() => {
                  setFormData((prev) => ({ ...prev, mfaCode: '' }));
                  navigate('/login', { replace: true, state: location.state });
                }}
                className="w-full text-sm text-gray-400 hover:text-white"
              >
                Back to login
              </button>
            )}
          </form>
        </div>

        {/* Footer */}
        <p className="mt-6 text-center text-sm text-gray-400">
          © 2024 OpenPAM. All rights reserved.
        </p>
      </div>
    </div>
  );
};
