import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar';

export default function AppLayout() {
  return (
    <div className="min-h-screen bg-app-bg lg:flex">
      <a href="#main-content" className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-[100] focus:rounded-lg focus:bg-emerald-300 focus:p-3 focus:text-slate-900">Skip to content</a>
      <Sidebar />
      <main id="main-content" tabIndex={-1} className="min-w-0 flex-1 px-4 py-8 outline-none sm:px-8 lg:px-10 lg:py-10 xl:px-12">
        <div className="mx-auto max-w-[1440px]"><Outlet /></div>
      </main>
    </div>
  );
}
