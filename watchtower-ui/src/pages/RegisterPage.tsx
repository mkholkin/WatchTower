import { useState, FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import AuthLayout from '../components/layout/AuthLayout';
import { mfaEnabled, authError } from '../api/auth';
import { useAuth } from '../context/AuthContext';

export default function RegisterPage() {
  const { register } = useAuth();
  const navigate = useNavigate();
  const [form, setForm] = useState({ login: '', password: '', email: '' });
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);
    try {
      await register(form.login, form.password, form.email);
      navigate(mfaEnabled ? '/login' : '/');
    } catch (err: any) {
      setError(authError(err));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <AuthLayout title="Start keeping watch" subtitle="Create an account to set up your monitoring workspace.">
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="username" className="block text-sm font-medium text-slate-400 mb-1.5">Username</label>
          <input
            id="username"
            autoComplete="username"
            type="text"
            required
            value={form.login}
            onChange={(e) => setForm({ ...form, login: e.target.value })}
            className="w-full px-3 py-2.5 bg-app-bg border border-border rounded-lg text-sm text-slate-100 placeholder-slate-600 focus:ring-2 focus:ring-emerald-500/30 focus:border-emerald-500/50 outline-none transition-all"
            placeholder="Choose a username"
          />
        </div>
        <div>
          <label htmlFor="password" className="block text-sm font-medium text-slate-400 mb-1.5">Password</label>
          <input
            id="password"
            autoComplete="new-password"
            type="password"
            required
            value={form.password}
            onChange={(e) => setForm({ ...form, password: e.target.value })}
            className="w-full px-3 py-2.5 bg-app-bg border border-border rounded-lg text-sm text-slate-100 placeholder-slate-600 focus:ring-2 focus:ring-emerald-500/30 focus:border-emerald-500/50 outline-none transition-all"
            placeholder="Choose a password"
          />
        </div>
        {mfaEnabled && <div>
          <label htmlFor="email" className="block text-sm text-slate-400 mb-2">Email</label>
          <input id="email" type="email" autoComplete="email" required value={form.email} onChange={e => setForm({ ...form, email: e.target.value })} className="w-full px-3 py-2.5 bg-app-bg border border-border rounded-lg text-slate-100" />
          <p className="mt-2 text-xs text-slate-400">Verification codes and recovery instructions will be sent here. Use a password of 1–72 UTF-8 bytes.</p>
        </div>}
        {error && (
          <div role="alert" className="bg-red-950 border border-red-900/50 text-red-400 px-4 py-2.5 rounded-lg text-sm">{error}</div>
        )}
        <button
          type="submit"
          disabled={submitting}
          className="primary-button w-full !py-3"
        >
          {submitting ? 'Creating account...' : 'Create Account'}
        </button>
      </form>
      <p className="text-center text-sm text-slate-500 mt-5">
        Already have an account?{' '}
        <Link to="/login" className="text-emerald-400 hover:text-emerald-300 transition-colors font-medium">Sign In</Link>
      </p>
    </AuthLayout>
  );
}
