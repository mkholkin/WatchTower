import { NavLink } from 'react-router-dom';
import { Activity, LayoutDashboard, Bell, Wrench, LogOut, ArrowUpRight } from 'lucide-react';
import { useAuth } from '../../context/AuthContext';

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Overview' },
  { to: '/alert-contacts', icon: Bell, label: 'Alert contacts' },
  { to: '/maintenance-windows', icon: Wrench, label: 'Maintenance' },
];

export default function Sidebar() {
  const { user, logout } = useAuth();
  return (
    <aside className="flex shrink-0 flex-col border-b border-border bg-sidebar-bg lg:sticky lg:top-0 lg:h-screen lg:w-60 lg:border-b-0 lg:border-r">
      <div className="flex items-center justify-between px-5 py-5 lg:px-6 lg:py-8">
        <NavLink to="/" className="flex items-center gap-3" aria-label="WatchTower overview">
          <span className="flex h-9 w-9 items-center justify-center rounded-xl border border-emerald-400/20 bg-emerald-400/10 text-emerald-300"><Activity size={21} /></span>
          <span className="text-lg font-semibold tracking-tight text-slate-100">WatchTower<span className="text-emerald-400">.</span></span>
        </NavLink>
        <button onClick={logout} className="rounded-lg p-2 text-slate-400 hover:bg-border lg:hidden" aria-label="Sign out"><LogOut size={18} /></button>
      </div>
      <nav aria-label="Main navigation" className="flex gap-1 overflow-x-auto px-3 pb-3 lg:flex-1 lg:flex-col lg:overflow-visible lg:px-4 lg:pt-5">
        <p className="eyebrow mb-3 hidden px-3 lg:block">Workspace</p>
        {navItems.map(({ to, icon: Icon, label }) => (
          <NavLink key={to} to={to} end={to === '/'} className={({ isActive }) =>
            `group flex shrink-0 items-center gap-2 rounded-lg px-3 py-3 text-xs font-medium transition-colors sm:text-sm lg:gap-3 ${isActive ? 'bg-emerald-400/10 text-emerald-300' : 'text-slate-400 hover:bg-white/5 hover:text-slate-100'}`
          }>
            <Icon size={17} aria-hidden="true" /><span>{label}</span>
          </NavLink>
        ))}
      </nav>
      <div className="hidden lg:block">
        <div className="mx-4 mb-6 rounded-xl border border-border bg-app-bg/50 p-4">
          <ArrowUpRight size={18} className="mb-3 text-emerald-300" />
          <p className="text-xs font-medium text-slate-200">A little peace of mind.</p>
          <p className="mt-2 text-xs leading-relaxed text-slate-500">Your endpoints, alerts, and downtime. All in one place.</p>
        </div>
        <div className="flex items-center gap-3 border-t border-border px-5 py-5">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-border bg-card-bg text-xs font-semibold uppercase text-slate-300">{user?.login?.[0] || '?'}</div>
          <div className="min-w-0 flex-1"><p className="truncate text-sm font-medium text-slate-200">{user?.login}</p><p className="mt-0.5 text-[11px] text-slate-500">Your workspace</p></div>
          <button onClick={logout} className="rounded-lg p-2 text-slate-500 hover:bg-border hover:text-slate-200" aria-label="Sign out"><LogOut size={16} /></button>
        </div>
      </div>
    </aside>
  );
}
