import api from './axios';
import { MonitorCheck, MonitorSLA, MonitorStatusEvent } from '../types';

export const getChecks = (monitorId: string, params?: Record<string, unknown>) =>
  api.get<MonitorCheck[]>(`/monitors/${monitorId}/checks`, { params });

export const getSLA = (monitorId: string, params?: Record<string, unknown>) =>
  api.get<MonitorSLA>(`/monitors/${monitorId}/sla`, { params });

export const getStatusHistory = (monitorId: string, params?: Record<string, unknown>) =>
  api.get<MonitorStatusEvent[]>(`/monitors/${monitorId}/status-history`, { params });
