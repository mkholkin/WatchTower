import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar';

export default function AppLayout() {
  return (
    <div className="flex min-h-screen bg-app-bg">
      <Sidebar />
      <main className="flex-1 p-8">
        <Outlet />
      </main>
    </div>
  );
}
