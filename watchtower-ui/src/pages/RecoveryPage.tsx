import { useState, FormEvent } from 'react';
import { Link } from 'react-router-dom';
import AuthLayout from '../components/layout/AuthLayout';
import { recover, reset, authError } from '../api/auth';
export default function RecoveryPage() {
  const [login, setLogin] = useState('');
  const [sent, setSent] = useState(false);
  const [token, setToken] = useState('');
  const [password, setPassword] = useState('');
  const [done, setDone] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  async function submit(e: FormEvent) {
    e.preventDefault(); setBusy(true); setError('');
    try {
      if (!sent) { await recover(login); setSent(true); }
      else { await reset(login, token, password); setToken(''); setPassword(''); setDone(true); }
    } catch (err) { setError(authError(err)); } finally { setBusy(false); }
  }
  const input = 'w-full px-3 py-2.5 bg-app-bg border border-border rounded-lg text-slate-100';
  return <AuthLayout title="Recover your account" subtitle={done ? 'Password updated. Sign in again with your new password and email code.' : sent ? 'If the account exists, recovery instructions were sent to its email. The token expires in ten minutes.' : 'Enter your username to receive recovery instructions.'}>
    {!done && <form onSubmit={submit} className="space-y-4">
      <label className="block text-sm text-slate-400">Username<input className={input} required autoComplete="username" value={login} disabled={sent} onChange={e => setLogin(e.target.value)} /></label>
      {sent && <><label className="block text-sm text-slate-400">Recovery token<input className={input} required autoComplete="off" value={token} onChange={e => setToken(e.target.value)} /></label>
      <label className="block text-sm text-slate-400">New password<input className={input} required type="password" autoComplete="new-password" value={password} onChange={e => setPassword(e.target.value)} /></label><p className="text-xs text-slate-400">Use a different password of 1–72 UTF-8 bytes.</p></>}
      {error && <p role="alert" className="text-red-400 text-sm">{error}</p>}
      <button className="primary-button w-full" disabled={busy}>{busy ? 'Please wait...' : sent ? 'Reset password' : 'Send recovery instructions'}</button>
      {sent && <button type="button" disabled={busy} className="text-emerald-400 text-sm" onClick={() => { setSent(false); setToken(''); setError(''); }}>Start again</button>}
    </form>}
    <Link className="block mt-5 text-sm text-emerald-400" to="/login">Back to sign in</Link>
  </AuthLayout>;
}
