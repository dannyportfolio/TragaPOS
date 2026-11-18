import React, { useEffect, useRef, useState } from 'react';
import {
  ShieldCheck,
  QrCode,
  CheckCircle2,
  XCircle,
  ArrowRight,
  ScanLine,
  Camera,
  VideoOff,
} from 'lucide-react';
import { api } from '../../services/api';
import { toast } from '../../store/useToastStore';

interface ScanResult {
  status: 'GRANTED' | 'DENIED';
  ticketId?: string;
  pax?: number;
  zone?: string;
  message?: string;
  time: string;
}

export const GateScannerView: React.FC = () => {
  const [qrInput, setQrInput] = useState('');
  const [gateZone, setGateZone] = useState('MAIN_GATE');
  const [isLoading, setIsLoading] = useState(false);
  const [lastScanResult, setLastScanResult] = useState<ScanResult | null>(null);
  const [scanHistory, setScanHistory] = useState<ScanResult[]>([]);

  const [cameraActive, setCameraActive] = useState(false);
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const streamRef = useRef<MediaStream | null>(null);

  const stopCamera = () => {
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((t) => t.stop());
      streamRef.current = null;
    }
    setCameraActive(false);
  };

  useEffect(() => {
    return () => stopCamera();
  }, []);

  const verifyQR = async (rawToken: string) => {
    const trimmed = rawToken.trim();
    if (!trimmed || isLoading) return;

    setIsLoading(true);
    try {
      const res = await api.redeemTicket({
        qr_string: trimmed,
        gate_zone: gateZone,
        device_id: 'TURNSTILE_LANE_01',
      });

      const result: ScanResult = {
        status: 'GRANTED',
        ticketId: res.ticket_id,
        pax: res.pax_redeemed,
        zone: res.gate_zone,
        time: new Date().toLocaleTimeString(),
      };
      setLastScanResult(result);
      setScanHistory((prev) => [result, ...prev.slice(0, 7)]);
      setQrInput('');
    } catch (err: any) {
      const result: ScanResult = {
        status: 'DENIED',
        zone: gateZone,
        message: err.message,
        time: new Date().toLocaleTimeString(),
      };
      setLastScanResult(result);
      setScanHistory((prev) => [result, ...prev.slice(0, 7)]);
    } finally {
      setIsLoading(false);
    }
  };

  const handleScanSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await verifyQR(qrInput);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      verifyQR(qrInput);
    }
  };

  const startCamera = async () => {
    const Detector = (window as any).BarcodeDetector;
    if (!Detector) {
      toast.info('Browser camera BarcodeDetector is not supported here. Use a 2D scanner or paste token.');
      return;
    }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: 'environment' },
      });
      streamRef.current = stream;
      setCameraActive(true);
      if (videoRef.current) {
        videoRef.current.srcObject = stream;
        await videoRef.current.play();
      }
      const detector = new Detector({ formats: ['qr_code'] });
      const pollFrame = async () => {
        if (!streamRef.current || !videoRef.current) return;
        try {
          const codes = await detector.detect(videoRef.current);
          if (codes && codes.length > 0 && codes[0].rawValue) {
            stopCamera();
            setQrInput(codes[0].rawValue);
            await verifyQR(codes[0].rawValue);
            return;
          }
        } catch {
          // ignore frame detection error
        }
        if (streamRef.current) {
          requestAnimationFrame(pollFrame);
        }
      };
      requestAnimationFrame(pollFrame);
    } catch (err: any) {
      toast.error(err.message || 'Unable to access camera');
      stopCamera();
    }
  };

  return (
    <div className="flex-1 flex flex-col h-full bg-slate-50 dark:bg-slate-950 p-3 sm:p-4 lg:p-6 overflow-y-auto select-none transition-colors">
      {/* Top Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-5">
        <div className="flex items-center gap-2.5">
          <div className="w-10 h-10 rounded-xl bg-purple-50 dark:bg-purple-500/10 text-purple-700 dark:text-purple-400 border border-purple-200 dark:border-purple-500/20 flex items-center justify-center shrink-0">
            <ScanLine className="w-5 h-5" />
          </div>
          <div>
            <h2 className="font-bold text-slate-900 dark:text-slate-100 text-sm sm:text-base flex flex-wrap items-center gap-2">
              Gate Turnstile Pass Validator
              <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-emerald-50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-500/30">
                100% OFFLINE ED25519
              </span>
            </h2>
            <p className="text-xs text-slate-600 dark:text-slate-400">Zero network access mathematical verification</p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <label htmlFor="gate-zone-select" className="text-xs font-medium text-slate-600 dark:text-slate-400">Gate Zone:</label>
          <select
            id="gate-zone-select"
            value={gateZone}
            onChange={(e) => setGateZone(e.target.value)}
            className="min-h-[44px] px-3 py-1.5 rounded-xl bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-200 text-xs font-medium focus:outline-none focus:border-purple-500"
          >
            <option value="MAIN_GATE">MAIN_GATE (Turnstile 1)</option>
            <option value="VIP_GATE">VIP_GATE (Turnstile 2)</option>
            <option value="WATERPARK_ENTRY">WATERPARK_ENTRY</option>
          </select>
        </div>
      </div>

      <div className="flex-1 grid grid-cols-1 lg:grid-cols-2 gap-5 sm:gap-6">
        {/* Left: QR Scan Input Form */}
        <div className="bg-white dark:bg-slate-900/60 rounded-2xl border border-slate-200 dark:border-slate-800/80 p-4 sm:p-6 flex flex-col justify-between gap-4 shadow-sm">
          <div className="space-y-4">
            <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-xs text-slate-600 dark:text-slate-400 leading-relaxed flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <span>
                Scan with a 2D optical scanner (auto-submits on Enter), use camera, or paste the raw token.
              </span>
              <button
                type="button"
                onClick={cameraActive ? stopCamera : startCamera}
                className={`min-h-[44px] px-3.5 py-2 rounded-xl text-xs font-semibold flex items-center justify-center gap-1.5 shrink-0 transition-colors ${
                  cameraActive
                    ? 'bg-rose-50 dark:bg-rose-500/20 text-rose-700 dark:text-rose-300 border border-rose-300 dark:border-rose-500/30'
                    : 'bg-white dark:bg-slate-800 hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-200 border border-slate-300 dark:border-slate-700'
                }`}
              >
                {cameraActive ? <VideoOff className="w-4 h-4" /> : <Camera className="w-4 h-4" />}
                <span>{cameraActive ? 'Stop Camera' : 'Camera Scan'}</span>
              </button>
            </div>

            {cameraActive && (
              <div className="rounded-xl overflow-hidden border border-purple-400 dark:border-purple-500/40 bg-black">
                <video ref={videoRef} className="w-full max-h-56 object-cover" muted playsInline />
              </div>
            )}

            <form onSubmit={handleScanSubmit} className="space-y-3">
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                  Raw QR Ticket String (<code className="text-indigo-700 dark:text-indigo-400 font-mono">payload.signature</code>)
                </label>
                <div className="relative">
                  <QrCode className="w-4 h-4 absolute left-3.5 top-3 text-slate-500 dark:text-slate-400" />
                  <textarea
                    value={qrInput}
                    onChange={(e) => setQrInput(e.target.value)}
                    onKeyDown={handleKeyDown}
                    rows={4}
                    placeholder="eyJ0aWNrZXRfaWQiOi... .4k8s9f..."
                    className="w-full pl-10 pr-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs sm:text-sm focus:outline-none focus:border-purple-500 font-mono resize-none"
                    required
                  />
                </div>
              </div>

              <button
                type="submit"
                disabled={isLoading}
                className="w-full min-h-[44px] py-3 rounded-xl bg-purple-600 hover:bg-purple-700 dark:hover:bg-purple-500 text-white font-semibold text-xs sm:text-sm flex items-center justify-center gap-2 transition-all shadow-lg shadow-purple-600/20 disabled:opacity-50"
              >
                <span>{isLoading ? 'Verifying Ed25519 Math...' : 'Verify Pass & Trigger Gate'}</span>
                <ArrowRight className="w-4 h-4" />
              </button>
            </form>
          </div>

          {scanHistory.length > 0 && (
            <div className="pt-3 border-t border-slate-200 dark:border-slate-800 space-y-2">
              <div className="text-[11px] font-semibold uppercase tracking-wider text-slate-600 dark:text-slate-400">
                Recent Gate Events
              </div>
              <div className="space-y-1.5 max-h-32 overflow-y-auto">
                {scanHistory.map((ev, idx) => (
                  <div
                    key={`${ev.time}-${idx}`}
                    className="px-3 py-1.5 rounded-lg bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800/80 text-[11px] flex items-center justify-between gap-2"
                  >
                    <span
                      className={`font-bold ${
                        ev.status === 'GRANTED' ? 'text-emerald-700 dark:text-emerald-400' : 'text-rose-700 dark:text-rose-400'
                      }`}
                    >
                      {ev.status}
                    </span>
                    <span className="text-slate-600 dark:text-slate-400 truncate flex-1">
                      {ev.status === 'GRANTED' ? `${ev.pax} pax • ${ev.zone}` : ev.message}
                    </span>
                    <span className="font-mono text-slate-500 dark:text-slate-400">{ev.time}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          <div className="pt-3 border-t border-slate-200 dark:border-slate-800 text-[11px] text-slate-600 dark:text-slate-400 flex items-center justify-between">
            <span className="flex items-center gap-1.5">
              <ShieldCheck className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400" />
              Cryptographic Turnstile Relay Ready
            </span>
            <span>Timeout: 500ms</span>
          </div>
        </div>

        {/* Right: Turnstile Physical Gate Simulator Status */}
        <div className="bg-white dark:bg-slate-900/60 rounded-2xl border border-slate-200 dark:border-slate-800/80 p-6 flex flex-col items-center justify-center text-center min-h-[280px] shadow-sm">
          {!lastScanResult ? (
            <div className="space-y-3">
              <div className="w-20 h-20 rounded-full bg-slate-100 dark:bg-slate-800 border-2 border-slate-300 dark:border-slate-700 flex items-center justify-center mx-auto text-slate-500 dark:text-slate-400">
                <ScanLine className="w-8 h-8 animate-pulse" />
              </div>
              <h3 className="font-semibold text-slate-900 dark:text-slate-200 text-sm">Turnstile Standby</h3>
              <p className="text-xs text-slate-600 dark:text-slate-400 max-w-xs">
                Awaiting ticket barcode scan. Physical relay barrier is closed.
              </p>
            </div>
          ) : lastScanResult.status === 'GRANTED' ? (
            <div className="space-y-4 w-full flex flex-col items-center">
              <div className="w-24 h-24 rounded-full bg-emerald-50 dark:bg-emerald-500/20 border-4 border-emerald-500 flex items-center justify-center mx-auto text-emerald-600 dark:text-emerald-400 shadow-xl shadow-emerald-500/20">
                <CheckCircle2 className="w-12 h-12" />
              </div>
              <div>
                <h3 className="text-xl font-bold text-emerald-700 dark:text-emerald-400 tracking-wide">ACCESS GRANTED</h3>
                <p className="text-xs text-slate-700 dark:text-slate-300 mt-1">Barrier Relay Open (Gate Turnstile Unlocked)</p>
              </div>

              <div className="p-4 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-xs font-mono text-left space-y-1.5 w-full max-w-xs">
                <div className="flex justify-between text-slate-600 dark:text-slate-400">
                  <span>Zone:</span>
                  <span className="text-slate-900 dark:text-slate-100 font-bold">{lastScanResult.zone}</span>
                </div>
                <div className="flex justify-between text-slate-600 dark:text-slate-400">
                  <span>Pax Allowed:</span>
                  <span className="text-emerald-700 dark:text-emerald-400 font-bold">{lastScanResult.pax} Person(s)</span>
                </div>
                <div className="flex justify-between text-slate-600 dark:text-slate-400">
                  <span>Ticket ID:</span>
                  <span className="text-slate-800 dark:text-slate-300">{lastScanResult.ticketId?.substring(0, 12)}...</span>
                </div>
                <div className="flex justify-between text-slate-600 dark:text-slate-400">
                  <span>Scan Time:</span>
                  <span className="text-slate-800 dark:text-slate-300">{lastScanResult.time}</span>
                </div>
              </div>
            </div>
          ) : (
            <div className="space-y-4 w-full flex flex-col items-center">
              <div className="w-24 h-24 rounded-full bg-rose-50 dark:bg-rose-500/20 border-4 border-rose-500 flex items-center justify-center mx-auto text-rose-600 dark:text-rose-400 shadow-xl shadow-rose-500/20">
                <XCircle className="w-12 h-12" />
              </div>
              <div>
                <h3 className="text-xl font-bold text-rose-700 dark:text-rose-400 tracking-wide">ACCESS DENIED</h3>
                <p className="text-xs text-slate-700 dark:text-slate-300 mt-1">Barrier Remains Locked</p>
              </div>

              <div className="p-4 rounded-xl bg-rose-50 dark:bg-slate-950 border border-rose-200 dark:border-rose-500/30 text-xs text-rose-800 dark:text-rose-300 text-left w-full max-w-xs">
                <div className="font-semibold mb-1">Rejection Reason:</div>
                <div className="font-mono text-[11px] leading-relaxed break-words">{lastScanResult.message}</div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
