import { useState, FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import AuthLayout from '../components/layout/AuthLayout';
import { useAuth } from '../context/AuthContext';

export default function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [form, setForm] = useState({ login: '', password: '' });
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);
    try {
      await login(form.login, form.password);
      navigate('/');
    } catch (err: any) {
      setError(err.response?.data?.message || 'Login failed');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <AuthLayout title="Welcome back" subtitle="Sign in to see how your services are doing.">
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
        {error && (
          <div className="bg-red-950 border border-red-900/50 text-red-400 px-4 py-2.5 rounded-lg text-sm">{error}</div>
        )}
        <button
          type="submit"
          disabled={submitting}
          className="primary-button w-full !py-3"
        >
          {submitting ? 'Signing in...' : 'Sign In'}
        </button>
      </form>
      <p className="text-center text-sm text-slate-500 mt-5">
        Don't have an account?{' '}
        <Link to="/register" className="text-emerald-400 hover:text-emerald-300 transition-colors font-medium">Sign Up</Link>
      </p>
    </AuthLayout>
  );
}
