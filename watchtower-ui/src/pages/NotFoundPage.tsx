import { Link } from 'react-router-dom';

export default function NotFoundPage() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-app-bg">
      <div className="text-center">
        <h1 className="text-8xl font-bold text-slate-800 mb-4">404</h1>
        <p className="text-slate-400 mb-8">Page not found</p>
        <Link
          to="/"
          className="inline-flex px-4 py-2.5 bg-emerald-500 text-slate-900 rounded-lg text-sm font-semibold hover:bg-emerald-400 transition-all"
        >
          Go to Dashboard
        </Link>
      </div>
    </div>
  );
}
