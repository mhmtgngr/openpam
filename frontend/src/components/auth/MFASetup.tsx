import React, { useState } from 'react';
import { Shield, Copy, Check } from 'lucide-react';
import { authApi } from '@/api/auth';
import { Button } from '@/components/common';
import { Input } from '@/components/common';
import toast from 'react-hot-toast';

interface MFASetupProps {
  onComplete: () => void;
  onCancel?: () => void;
}

export const MFASetup: React.FC<MFASetupProps> = ({ onComplete, onCancel }) => {
  const [step, setStep] = useState<'setup' | 'verify'>('setup');
  const [qrCodeUrl, setQrCodeUrl] = useState('');
  const [secret, setSecret] = useState('');
  const [backupCodes, setBackupCodes] = useState<string[]>([]);
  const [code, setCode] = useState('');
  const [copied, setCopied] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const setupTOTP = async () => {
    setIsLoading(true);
    try {
      const response = await authApi.setupTOTP();
      setQrCodeUrl(response.qr_code_url);
      setSecret(response.secret);
      setBackupCodes(response.backup_codes);
      setStep('verify');
    } catch {
      // Error handled by interceptor
    } finally {
      setIsLoading(false);
    }
  };

  const verifyTOTP = async () => {
    if (!code || code.length !== 6) {
      toast.error('Please enter a 6-digit code');
      return;
    }

    setIsLoading(true);
    try {
      await authApi.verifyTOTP(secret, code);
      toast.success('MFA enabled successfully');
      onComplete();
    } catch {
      // Error handled by interceptor
    } finally {
      setIsLoading(false);
    }
  };

  const copySecret = () => {
    navigator.clipboard.writeText(secret);
    setCopied(true);
    toast.success('Secret copied to clipboard');
    setTimeout(() => setCopied(false), 2000);
  };

  if (step === 'setup') {
    return (
      <div className="mx-auto max-w-md">
        <div className="text-center">
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-primary-600/20">
            <Shield className="h-8 w-8 text-primary-500" />
          </div>
          <h2 className="text-2xl font-bold text-white">Set Up Two-Factor Authentication</h2>
          <p className="mt-2 text-gray-400">
            Add an extra layer of security to your account
          </p>
        </div>

        <div className="mt-8 card">
          <h3 className="text-lg font-medium text-white">Authenticator App</h3>
          <p className="mt-2 text-sm text-gray-400">
            We recommend using an authenticator app like Google Authenticator,
            Authy, or 1Password.
          </p>

          <div className="mt-6 space-y-4">
            <div className="flex items-start gap-3 rounded-md bg-gray-800/50 p-4">
              <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary-600 text-xs font-bold text-white">
                1
              </div>
              <p className="text-sm text-gray-300">
                Install an authenticator app on your mobile device
              </p>
            </div>

            <div className="flex items-start gap-3 rounded-md bg-gray-800/50 p-4">
              <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary-600 text-xs font-bold text-white">
                2
              </div>
              <p className="text-sm text-gray-300">
                Scan the QR code or enter the secret key manually
              </p>
            </div>

            <div className="flex items-start gap-3 rounded-md bg-gray-800/50 p-4">
              <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary-600 text-xs font-bold text-white">
                3
              </div>
              <p className="text-sm text-gray-300">
                Enter the 6-digit code from your app to verify
              </p>
            </div>
          </div>

          <div className="mt-6 flex gap-3">
            <Button
              variant="secondary"
              className="flex-1"
              onClick={onCancel}
              disabled={isLoading}
            >
              Skip for now
            </Button>
            <Button
              variant="primary"
              className="flex-1"
              onClick={setupTOTP}
              isLoading={isLoading}
            >
              Continue
            </Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-md">
      <div className="text-center">
        <h2 className="text-2xl font-bold text-white">Scan QR Code</h2>
        <p className="mt-2 text-gray-400">
          Scan this QR code with your authenticator app
        </p>
      </div>

      <div className="mt-8 card">
        {/* QR Code */}
        <div className="flex justify-center">
          <img
            src={qrCodeUrl}
            alt="QR Code"
            className="h-64 w-64 rounded-lg bg-white p-4"
          />
        </div>

        {/* Secret Key */}
        <div className="mt-6">
          <label className="mb-2 block text-sm font-medium text-gray-300">
            Or enter this code manually:
          </label>
          <div className="flex gap-2">
            <input
              type="text"
              value={secret}
              readOnly
              className="input font-mono text-sm"
            />
            <Button
              variant="secondary"
              size="sm"
              onClick={copySecret}
              leftIcon={copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
            >
              {copied ? 'Copied' : 'Copy'}
            </Button>
          </div>
        </div>

        {/* Verification Code */}
        <div className="mt-6">
          <Input
            label="Enter the 6-digit code"
            type="text"
            placeholder="123456"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            maxLength={6}
            pattern="[0-9]*"
            inputMode="numeric"
            autoFocus
          />
        </div>

        {/* Backup Codes */}
        <details className="mt-6">
          <summary className="cursor-pointer text-sm text-gray-400 hover:text-white">
            View backup codes
          </summary>
          <div className="mt-3 rounded-md bg-gray-800/50 p-4">
            <p className="mb-2 text-xs text-gray-400">
              Save these backup codes in a safe place. You can use them to access your account
              if you lose access to your authenticator app.
            </p>
            <div className="grid grid-cols-2 gap-2 font-mono text-xs">
              {backupCodes.map((code, index) => (
                <div key={index} className="rounded bg-gray-800 px-2 py-1 text-gray-300">
                  {code}
                </div>
              ))}
            </div>
          </div>
        </details>

        <div className="mt-6 flex gap-3">
          <Button
            variant="secondary"
            className="flex-1"
            onClick={() => {
              setStep('setup');
              setCode('');
            }}
            disabled={isLoading}
          >
            Back
          </Button>
          <Button
            variant="primary"
            className="flex-1"
            onClick={verifyTOTP}
            isLoading={isLoading}
          >
            Verify & Enable
          </Button>
        </div>
      </div>
    </div>
  );
};
