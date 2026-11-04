import { useState, FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import AuthLayout from '../components/layout/AuthLayout';
import { mfaEnabled, authError } from '../api/auth';
import { useAuth } from '../context/AuthContext';

export default function LoginPage() {
  const { login, verifyLogin } = useAuth();
  const navigate = useNavigate();
  const [form, setForm] = useState({ login: '', password: '' });
  const [challenge, setChallenge] = useState('');
  const [code, setCode] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);
    try {
      if (challenge) {
        const rotate = await verifyLogin(form.login, challenge, code);
        navigate(rotate ? '/change-password' : '/');
      } else {
        const next = await login(form.login, form.password);
        setForm({ ...form, password: '' });
        if (next) setChallenge(next); else navigate('/');
      }
    } catch (err: any) {
      setError(authError(err));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <AuthLayout title={challenge ? "Check your email" : "Welcome back"} subtitle={challenge ? "Enter the six-digit code sent to your registered email. It expires in five minutes." : "Sign in to see how your services are doing."}>
      <form onSubmit={handleSubmit} className="space-y-4">
        {!challenge && <><div>
          <label htmlFor="username" className="block text-sm font-medium text-slate-400 mb-1.5">Username</label>
          <input
            id="username"
            autoComplete="username"
            type="text"
            required
            value={form.login}
            onChange={(e) => setForm({ ...form, login: e.target.value })}
            className="w-full px-3 py-2.5 bg-app-bg border border-border rounded-lg text-sm text-slate-100 placeholder-slate-600 focus:ring-2 focus:ring-emerald-500/30 focus:border-emerald-500/50 outline-none transition-all"
            placeholder="Enter username"
          />
        </div>
        <div>
          <label htmlFor="password" className="block text-sm font-medium text-slate-400 mb-1.5">Password</label>
          <input
            id="password"
            autoComplete="current-password"
            type="password"
            required
            value={form.password}
            onChange={(e) => setForm({ ...form, password: e.target.value })}
            className="w-full px-3 py-2.5 bg-app-bg border border-border rounded-lg text-sm text-slate-100 placeholder-slate-600 focus:ring-2 focus:ring-emerald-500/30 focus:border-emerald-500/50 outline-none transition-all"
            placeholder="Enter password"
          />
        </div>
        </>}
        {challenge && <div>
          <label htmlFor="code" className="block text-sm text-slate-400 mb-2">Verification code</label>
          <input id="code" autoFocus autoComplete="one-time-code" inputMode="numeric" pattern="[0-9]{6}" maxLength={6} required value={code} onChange={e => setCode(e.target.value.replace(/\D/g, ''))} className="w-full px-3 py-3 bg-app-bg border border-border rounded-lg text-slate-100 tracking-[0.5em] text-center" />
        </div>}
        {challenge && process.env.REACT_APP_MAILBOX_URL && <p className="text-sm text-slate-400">
          This local demo delivers codes to a test mailbox, not your external inbox.{' '}
          <a className="text-emerald-400 underline" href={process.env.REACT_APP_MAILBOX_URL} target="_blank" rel="noreferrer">Open verification email</a>
        </p>}
        {error && (
          <div role="alert" className="bg-red-950 border border-red-900/50 text-red-400 px-4 py-2.5 rounded-lg text-sm">{error}</div>
        )}
        <button
          type="submit"
          disabled={submitting}
          className="primary-button w-full !py-3"
        >
          {submitting ? 'Please wait...' : challenge ? 'Verify code' : mfaEnabled ? 'Send verification code' : 'Sign In'}
        </button>
      </form>
      {challenge && <button type="button" disabled={submitting} className="mt-4 text-sm text-emerald-400" onClick={() => { setChallenge(''); setCode(''); setError(''); }}>Start again / request a new code</button>}
      {mfaEnabled && <p className="mt-4 text-sm"><Link to="/recover" className="text-emerald-400">Recover your account</Link></p>}
      <p className="text-center text-sm text-slate-500 mt-5">
        Don't have an account?{' '}
        <Link to="/register" className="text-emerald-400 hover:text-emerald-300 transition-colors font-medium">Sign Up</Link>
      </p>
    </AuthLayout>
  );
}
