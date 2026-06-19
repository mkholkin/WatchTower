import api from './axios';
import { Monitor } from '../types';

export const getMonitors = () => api.get<Monitor[]>('/monitors');

export const getMonitor = (id: string) => api.get<Monitor>(`/monitors/${id}`);

export const createMonitor = (data: Record<string, unknown>) =>
  api.post<Monitor>('/monitors', data);

export const updateMonitor = (id: string, data: Record<string, unknown>) =>
  api.patch<Monitor>(`/monitors/${id}`, data);

export const deleteMonitor = (id: string) => api.delete(`/monitors/${id}`);

export const enableMonitor = (id: string) => api.post(`/monitors/${id}/enable`);

export const disableMonitor = (id: string) => api.post(`/monitors/${id}/disable`);

export const addAlertContact = (monitorId: string, alertContactId: string) =>
  api.post(`/monitors/${monitorId}/alert-contacts`, { alert_contact_id: alertContactId });

export const removeAlertContact = (monitorId: string, alertContactId: string) =>
  api.delete(`/monitors/${monitorId}/alert-contacts`, {
    data: { alert_contact_id: alertContactId },
  });
