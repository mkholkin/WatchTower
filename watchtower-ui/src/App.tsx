import { BrowserRouter, Routes, Route, Navigate, Outlet } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import { LoadingSpinner } from './components/shared/LoadingSpinner';
import AppLayout from './components/layout/AppLayout';
import LoginPage from './pages/LoginPage';
import RegisterPage from './pages/RegisterPage';
import DashboardPage from './pages/DashboardPage';
import MonitorDetailPage from './pages/MonitorDetailPage';
import AlertContactsPage from './pages/AlertContactsPage';
import MaintenanceWindowsPage from './pages/MaintenanceWindowsPage';
import NotFoundPage from './pages/NotFoundPage';

function ProtectedRoute() {
  const { user, loading } = useAuth();

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-app-bg">
        <LoadingSpinner />
      </div>
    );
  }

  if (!user) {
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route element={<ProtectedRoute />}>
            <Route element={<AppLayout />}>
              <Route path="/" element={<DashboardPage />} />
              <Route path="/monitors/:id" element={<MonitorDetailPage />} />
              <Route path="/alert-contacts" element={<AlertContactsPage />} />
              <Route path="/maintenance-windows" element={<MaintenanceWindowsPage />} />
            </Route>
          </Route>
          <Route path="*" element={<NotFoundPage />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}
