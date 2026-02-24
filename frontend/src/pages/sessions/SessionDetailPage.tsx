import React, { useState, useEffect, useRef } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Ban, Download, Eye, EyeOff, Maximize2, Minimize2, Loader2 } from 'lucide-react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';
import { sessionsApi } from '@/api/sessions';
import { Button, Card, Badge, Textarea } from '@/components/common';
import type { Session, SessionEvent } from '@/types';
import toast from 'react-hot-toast';
import clsx from 'clsx';

export const SessionDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const terminalRef = useRef<HTMLDivElement>(null);
  const terminalInstance = useRef<Terminal | null>(null);
  const fitAddon = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);

  const [showTerminateModal, setShowTerminateModal] = useState(false);
  const [terminateReason, setTerminateReason] = useState('');
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [isMonitoring, setIsMonitoring] = useState(false);

  // Fetch session details
  const { data: session, isLoading } = useQuery({
    queryKey: ['session', id],
    queryFn: () => sessionsApi.get(id!).then((res) => res.data),
    refetchInterval: (query) => {
      // Refetch every 5 seconds if session is active
      const session = query.state.data as Session | undefined;
      return session?.status === 'active' ? 5000 : false;
    },
  });

  // Fetch session events
  const { data: events } = useQuery({
    queryKey: ['session', id, 'events'],
    queryFn: () =>
      sessionsApi.getEvents(id!, { limit: 100 }).then((res) => res.data),
    enabled: !!id,
  });

  // Terminate mutation
  const terminateMutation = useMutation({
    mutationFn: ({ reason }: { reason?: string }) => sessionsApi.terminate(id!, reason),
    onSuccess: () => {
      toast.success('Session terminated');
      queryClient.invalidateQueries({ queryKey: ['session', id] });
      queryClient.invalidateQueries({ queryKey: ['sessions'] });
      setShowTerminateModal(false);
      setTerminateReason('');
      // Close WebSocket if open
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      setIsMonitoring(false);
    },
  });

  // Initialize terminal for active sessions
  useEffect(() => {
    if (session?.status === 'active' && terminalRef.current && !terminalInstance.current) {
      const terminal = new Terminal({
        theme: {
          background: '#1a1a2e',
          foreground: '#e4e4e7',
          cursor: '#e4e4e7',
          selection: 'rgba(255, 255, 255, 0.3)',
          black: '#1a1a2e',
          red: '#f87171',
          green: '#4ade80',
          yellow: '#fbbf24',
          blue: '#60a5fa',
          magenta: '#c084fc',
          cyan: '#22d3d8',
          white: '#e4e4e7',
          brightBlack: '#71717a',
          brightRed: '#fca5a5',
          brightGreen: '#86efac',
          brightYellow: '#fcd34d',
          brightBlue: '#93c5fd',
          brightMagenta: '#d8b4fe',
          brightCyan: '#67e8f9',
          brightWhite: '#ffffff',
        } as any,
        fontSize: 14,
        fontFamily: 'Monaco, "Cascadia Code", "Ubuntu Mono", monospace',
        cursorBlink: true,
        cursorStyle: 'block',
        scrollback: 1000,
      });

      const fitAddonInstance = new FitAddon();
      terminal.loadAddon(fitAddonInstance);

      terminal.open(terminalRef.current);
      fitAddonInstance.fit();

      terminalInstance.current = terminal;
      fitAddon.current = fitAddonInstance;

      // Handle window resize
      const handleResize = () => {
        fitAddonInstance?.fit();
      };
      window.addEventListener('resize', handleResize);

      return () => {
        window.removeEventListener('resize', handleResize);
        terminal.dispose();
        terminalInstance.current = null;
        fitAddon.current = null;
      };
    }
  }, [session?.status, terminalRef]);

  // Connect to WebSocket for monitoring
  const startMonitoring = () => {
    if (!session?.id || !terminalInstance.current) return;

    const wsUrl = (session as any)?.monitoring_url || `ws://localhost:8501/ws/sessions/${session.id}/monitor`;
    const token = localStorage.getItem('access_token');

    const ws = new WebSocket(`${wsUrl}?token=${token}`);

    ws.onopen = () => {
      toast.success('Connected to session');
      setIsMonitoring(true);
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type === 'output') {
          terminalInstance.current?.write(data.data);
        } else if (data.type === 'status') {
          console.log('Session status:', data.status);
        }
      } catch {
        // If not JSON, write directly as terminal output
        terminalInstance.current?.write(event.data);
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      toast.error('Connection error');
      setIsMonitoring(false);
    };

    ws.onclose = () => {
      toast('Session connection closed', { icon: 'ℹ️' });
      setIsMonitoring(false);
    };

    wsRef.current = ws;
  };

  const stopMonitoring = () => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setIsMonitoring(false);
  };

  const handleTerminate = () => {
    terminateMutation.mutate({ reason: terminateReason || undefined });
  };

  const downloadRecording = async () => {
    try {
      const response = await sessionsApi.getRecordingUrl(id!);
      const url = response.data.url;
      const a = document.createElement('a');
      a.href = url;
      a.download = `session-${id}.cast`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      toast.success('Recording downloaded');
    } catch (error) {
      toast.error('Failed to download recording');
    }
  };

  const toggleFullscreen = () => {
    setIsFullscreen(!isFullscreen);
    setTimeout(() => {
      fitAddon.current?.fit();
    }, 100);
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[50vh]">
        <Loader2 className="h-8 w-8 animate-spin text-primary-500" />
      </div>
    );
  }

  if (!session) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[50vh] gap-4">
        <h2 className="text-xl font-semibold text-white">Session not found</h2>
        <Link to="/sessions">
          <Button>Back to Sessions</Button>
        </Link>
      </div>
    );
  }

  const isActive = session.status === 'active';
  const hasRecording = session.recording_url && session.status !== 'active';

  return (
    <div className={clsx('space-y-6', isFullscreen && 'fixed inset-0 z-50 bg-gray-900 p-4')}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          {!isFullscreen && (
            <Link to="/sessions">
              <Button variant="ghost" size="sm" leftIcon={<ArrowLeft className="h-4 w-4" />}>
                Back
              </Button>
            </Link>
          )}
          <div>
            <h1 className="text-2xl font-bold text-white">Session {id?.slice(0, 8)}</h1>
            <p className="mt-1 text-sm text-gray-400">
              {session.user?.first_name} {session.user?.last_name} → {session.target?.name}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {isActive && !isMonitoring && session.monitoring_enabled && (
            <Button
              variant="secondary"
              size="sm"
              leftIcon={<Eye className="h-4 w-4" />}
              onClick={startMonitoring}
            >
              Monitor
            </Button>
          )}
          {isMonitoring && (
            <Button
              variant="secondary"
              size="sm"
              leftIcon={<EyeOff className="h-4 w-4" />}
              onClick={stopMonitoring}
            >
              Stop Monitoring
            </Button>
          )}
          {hasRecording && (
            <Button
              variant="secondary"
              size="sm"
              leftIcon={<Download className="h-4 w-4" />}
              onClick={downloadRecording}
            >
              Download
            </Button>
          )}
          {isActive && session.can_terminate && (
            <Button
              variant="danger"
              size="sm"
              leftIcon={<Ban className="h-4 w-4" />}
              onClick={() => setShowTerminateModal(true)}
            >
              Terminate
            </Button>
          )}
          <Button
            variant="ghost"
            size="sm"
            onClick={toggleFullscreen}
            leftIcon={isFullscreen ? <Minimize2 className="h-4 w-4" /> : <Maximize2 className="h-4 w-4" />}
          />
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Main content */}
        <div className={clsx('lg:col-span-3', isFullscreen && 'col-span-4')}>
          {/* Terminal / Recording Viewer */}
          <Card className={clsx(isActive && 'border-primary-500')}>
            <div className="card-body">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-white">
                  {isActive ? 'Live Session' : 'Session Recording'}
                </h3>
                {isActive && (
                  <div className="flex items-center gap-2">
                    <span className="relative flex h-2 w-2">
                      <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-success-400 opacity-75"></span>
                      <span className="relative inline-flex rounded-full h-2 w-2 bg-success-500"></span>
                    </span>
                    <span className="text-sm text-success-400">Live</span>
                  </div>
                )}
              </div>

              {isActive ? (
                <div
                  ref={terminalRef}
                  className="rounded-lg overflow-hidden bg-gray-900 border border-gray-700"
                  style={{ height: isFullscreen ? 'calc(100vh - 200px)' : '500px' }}
                />
              ) : (
                <div className="rounded-lg bg-gray-900 border border-gray-700 p-4 h-96 flex items-center justify-center">
                  {hasRecording ? (
                    <div className="text-center">
                      <p className="text-gray-400 mb-2">Recording available</p>
                      <Button
                        variant="secondary"
                        size="sm"
                        leftIcon={<Download className="h-4 w-4" />}
                        onClick={downloadRecording}
                      >
                        Download Recording
                      </Button>
                    </div>
                  ) : (
                    <p className="text-gray-400">No recording available</p>
                  )}
                </div>
              )}
            </div>
          </Card>

          {/* Session Events */}
          {events && events.data.length > 0 && (
            <Card>
              <div className="card-body">
                <h3 className="text-lg font-semibold text-white mb-4">Session Events</h3>
                <div className="space-y-2 max-h-64 overflow-y-auto">
                  {events.data.map((event: SessionEvent) => (
                    <div
                      key={event.id}
                      className="flex items-start gap-3 p-3 bg-gray-800 rounded-lg"
                    >
                      <Badge variant={event.type === 'warning' || event.type === 'error' ? 'danger' : 'neutral'} size="sm">
                        {event.type}
                      </Badge>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm text-white truncate">
                          {JSON.stringify(event.data)}
                        </p>
                        <p className="text-xs text-gray-400 mt-1">
                          {new Date(event.timestamp).toLocaleString()}
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </Card>
          )}
        </div>

        {/* Sidebar */}
        <div className="lg:col-span-1">
          <Card>
            <div className="card-body">
              <h3 className="text-lg font-semibold text-white mb-4">Session Details</h3>
              <dl className="space-y-3">
                <div>
                  <dt className="text-sm text-gray-400">Status</dt>
                  <dd className="mt-1">
                    <Badge
                      variant={
                        session.status === 'active'
                          ? 'success'
                          : session.status === 'terminated'
                            ? 'danger'
                            : 'neutral'
                      }
                    >
                      {session.status}
                    </Badge>
                  </dd>
                </div>

                <div>
                  <dt className="text-sm text-gray-400">Type</dt>
                  <dd className="mt-1 text-sm text-white uppercase">{session.type}</dd>
                </div>

                <div>
                  <dt className="text-sm text-gray-400">User</dt>
                  <dd className="mt-1 text-sm text-white">
                    {session.user?.first_name} {session.user?.last_name}
                    <p className="text-xs text-gray-400">{session.user?.email}</p>
                  </dd>
                </div>

                <div>
                  <dt className="text-sm text-gray-400">Target</dt>
                  <dd className="mt-1 text-sm text-white">
                    {session.target?.name}
                    <p className="text-xs text-gray-400">{session.target?.host}:{session.target?.port}</p>
                  </dd>
                </div>

                <div>
                  <dt className="text-sm text-gray-400">Client IP</dt>
                  <dd className="mt-1 text-sm text-white font-mono">{session.client_ip}</dd>
                </div>

                <div>
                  <dt className="text-sm text-gray-400">Started At</dt>
                  <dd className="mt-1 text-sm text-white">
                    {new Date(session.started_at).toLocaleString()}
                  </dd>
                </div>

                {session.ended_at && (
                  <div>
                    <dt className="text-sm text-gray-400">Ended At</dt>
                    <dd className="mt-1 text-sm text-white">
                      {new Date(session.ended_at).toLocaleString()}
                    </dd>
                  </div>
                )}

                {session.duration_seconds && (
                  <div>
                    <dt className="text-sm text-gray-400">Duration</dt>
                    <dd className="mt-1 text-sm text-white">
                      {Math.floor(session.duration_seconds / 60)}m {session.duration_seconds % 60}s
                    </dd>
                  </div>
                )}

                {session.terminated_by && (
                  <div>
                    <dt className="text-sm text-gray-400">Terminated By</dt>
                    <dd className="mt-1 text-sm text-white">{session.terminated_by}</dd>
                  </div>
                )}

                {session.terminated_reason && (
                  <div>
                    <dt className="text-sm text-gray-400">Termination Reason</dt>
                    <dd className="mt-1 text-sm text-white">{session.terminated_reason}</dd>
                  </div>
                )}

                {session.recording_size && (
                  <div>
                    <dt className="text-sm text-gray-400">Recording Size</dt>
                    <dd className="mt-1 text-sm text-white">
                      {(session.recording_size / 1024 / 1024).toFixed(2)} MB
                    </dd>
                  </div>
                )}
              </dl>
            </div>
          </Card>
        </div>
      </div>

      {/* Terminate Modal */}
      {showTerminateModal && (
        <div
          className="modal-backdrop"
          onClick={() => setShowTerminateModal(false)}
        >
          <div
            className="modal-content max-w-sm"
            onClick={(e) => e.stopPropagation()}
          >
            <h3 className="text-lg font-semibold text-white mb-4">Terminate Session</h3>
            <p className="mb-4 text-gray-300">
              Are you sure you want to terminate this session? This action cannot be undone.
            </p>
            <Textarea
              label="Reason (optional)"
              placeholder="Provide a reason for terminating this session..."
              value={terminateReason}
              onChange={(e) => setTerminateReason(e.target.value)}
              rows={3}
            />
            <div className="flex justify-end gap-3 mt-4">
              <Button
                variant="secondary"
                onClick={() => {
                  setShowTerminateModal(false);
                  setTerminateReason('');
                }}
              >
                Cancel
              </Button>
              <Button
                variant="danger"
                onClick={handleTerminate}
                isLoading={terminateMutation.isPending}
              >
                Terminate Session
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
