import { NavLink } from 'react-router-dom';
import { LayoutDashboard, Bell, Wrench, LogOut } from 'lucide-react'; // Removed Activity
import { useAuth } from '../../context/AuthContext';
// Import the logo image
import logoIcon from '../../assets/WatchTower-logo.png';

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/alert-contacts', icon: Bell, label: 'Alert Contacts' },
  { to: '/maintenance-windows', icon: Wrench, label: 'Maintenance' },
];

export default function Sidebar() {
  const { user, logout } = useAuth();

  return (
      <aside className="w-60 bg-sidebar-bg border-r border-border flex flex-col h-screen sticky top-0">
        <div className="px-5 py-4 border-b border-border">
          <div className="flex items-center gap-2.5">
            {/* Logo Container */}
            <div className="w-8 h-8 flex items-center justify-center">
              <img
                  src={logoIcon}
                  alt="WatchTower"
                  className="w-full h-full object-contain"
              />
            </div>
            <div>
              <h1 className="text-base font-bold text-slate-100 leading-tight">WatchTower</h1>
              <p className="text-[11px] text-slate-500 leading-tight">Monitoring</p>
            </div>
          </div>
        </div>

        <nav className="flex-1 p-3 space-y-0.5">
          <p className="px-3 py-2 text-[11px] font-semibold text-slate-600 uppercase tracking-widest">Menu</p>
          {navItems.map(({ to, icon: Icon, label }) => (
              <NavLink
                  key={to}
                  to={to}
                  end={to === '/'}
                  className={({ isActive }) =>
                      `flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                          isActive
                              ? 'bg-emerald-500/10 text-emerald-400'
                              : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'
                      }`
                  }
              >
                <Icon size={17} />
                {label}
              </NavLink>
          ))}
        </nav>

        <div className="p-3 border-t border-border">
          <div className="flex items-center gap-3 px-2 py-1.5">
            <div className="w-7 h-7 rounded-full bg-slate-600 flex items-center justify-center text-xs font-semibold text-slate-200 uppercase">
              {user?.login?.[0] || '?'}
            </div>
            <span className="text-sm text-slate-400 truncate flex-1">{user?.login}</span>
            <button
                onClick={logout}
                className="p-1 text-slate-600 hover:text-slate-300 rounded transition-colors"
                title="Logout"
            >
              <LogOut size={15} />
            </button>
          </div>
        </div>
      </aside>
  );
}