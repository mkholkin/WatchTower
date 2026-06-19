import api from './axios';
import { MaintenanceWindow } from '../types';

export const getMaintenanceWindows = () =>
  api.get<MaintenanceWindow[]>('/maintenance-windows');

export const getMaintenanceWindow = (id: string) =>
  api.get<MaintenanceWindow>(`/maintenance-windows/${id}`);

export const createMaintenanceWindow = (data: Record<string, unknown>) =>
  api.post<MaintenanceWindow>('/maintenance-windows', data);

export const updateMaintenanceWindow = (id: string, data: Record<string, unknown>) =>
  api.patch<MaintenanceWindow>(`/maintenance-windows/${id}`, data);

export const deleteMaintenanceWindow = (id: string) =>
  api.delete(`/maintenance-windows/${id}`);

export const addMonitorToWindow = (windowId: string, monitorId: string) =>
  api.post(`/maintenance-windows/${windowId}/monitors/`, { monitor_id: monitorId });

export const removeMonitorFromWindow = (windowId: string, monitorId: string) =>
  api.delete(`/maintenance-windows/${windowId}/monitors/`, {
    data: { monitor_id: monitorId },
  });
