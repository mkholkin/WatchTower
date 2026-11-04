import { useState, FormEvent } from 'react';
import { Navigate, useNavigate } from 'react-router-dom';
import AuthLayout from '../components/layout/AuthLayout';
import { useAuth } from '../context/AuthContext';
import { changePassword, authError } from '../api/auth';
export default function ChangePasswordPage() {
  const { rotationToken, logout } = useAuth();
  const navigate = useNavigate();
  const [current, setCurrent] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  if (!rotationToken) return <Navigate to="/login" replace />;
  async function submit(e: FormEvent) {
    e.preventDefault(); setBusy(true); setError('');
    try { await changePassword(rotationToken, current, password); logout(); navigate('/login'); }
    catch (err) { setError(authError(err)); } finally { setBusy(false); }
  }
  const input = 'w-full px-3 py-2.5 bg-app-bg border border-border rounded-lg text-slate-100';
  return <AuthLayout title="Update your password" subtitle="Your password has expired. Choose a different password of 1–72 UTF-8 bytes, then sign in again.">
    <form onSubmit={submit} className="space-y-4">
      <label className="block text-sm text-slate-400">Current password<input className={input} type="password" required autoComplete="current-password" value={current} onChange={e => setCurrent(e.target.value)} /></label>
      <label className="block text-sm text-slate-400">New password<input className={input} type="password" required autoComplete="new-password" value={password} onChange={e => setPassword(e.target.value)} /></label>
      {error && <p role="alert" className="text-red-400 text-sm">{error}</p>}
      <button className="primary-button w-full" disabled={busy}>{busy ? 'Updating...' : 'Update password'}</button>
      <button type="button" className="text-sm text-emerald-400" onClick={() => { logout(); navigate('/login'); }}>Cancel and sign out</button>
    </form>
  </AuthLayout>;
}
