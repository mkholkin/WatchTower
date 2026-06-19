import { useState, FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Activity } from 'lucide-react';
import { useAuth } from '../context/AuthContext';

export default function RegisterPage() {
  const { register } = useAuth();
  const navigate = useNavigate();
  const [form, setForm] = useState({ login: '', password: '' });
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);
    try {
      await register(form.login, form.password);
      navigate('/');
    } catch (err: any) {
      setError(err.response?.data?.message || 'Registration failed');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-app-bg">
      <div className="max-w-md w-full mx-4">
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-emerald-500/10 mb-4">
            <Activity size={24} className="text-emerald-400" />
          </div>
          <h1 className="text-2xl font-bold text-slate-100">WatchTower</h1>
          <p className="text-sm text-slate-500 mt-1">Create your account</p>
        </div>

        <div className="bg-card-bg rounded-xl border border-border shadow-xl shadow-black/20 p-6">
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-slate-400 mb-1.5">Username</label>
              <input
                type="text"
                required
                value={form.login}
                onChange={(e) => setForm({ ...form, login: e.target.value })}
                className="w-full px-3 py-2.5 bg-app-bg border border-border rounded-lg text-sm text-slate-100 placeholder-slate-600 focus:ring-2 focus:ring-emerald-500/30 focus:border-emerald-500/50 outline-none transition-all"
                placeholder="Choose a username"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-400 mb-1.5">Password</label>
              <input
                type="password"
                required
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
                className="w-full px-3 py-2.5 bg-app-bg border border-border rounded-lg text-sm text-slate-100 placeholder-slate-600 focus:ring-2 focus:ring-emerald-500/30 focus:border-emerald-500/50 outline-none transition-all"
                placeholder="Choose a password"
              />
            </div>
            {error && (
              <div className="bg-red-950 border border-red-900/50 text-red-400 px-4 py-2.5 rounded-lg text-sm">{error}</div>
            )}
            <button
              type="submit"
              disabled={submitting}
              className="w-full bg-emerald-500 text-slate-900 py-2.5 rounded-lg font-semibold text-sm hover:bg-emerald-400 disabled:opacity-50 transition-all"
            >
              {submitting ? 'Creating account...' : 'Create Account'}
            </button>
          </form>
          <p className="text-center text-sm text-slate-500 mt-5">
            Already have an account?{' '}
            <Link to="/login" className="text-emerald-400 hover:text-emerald-300 transition-colors font-medium">Sign In</Link>
          </p>
        </div>
      </div>
    </div>
  );
}
