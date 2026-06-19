import api from './axios';
import { AlertContact } from '../types';

export const getAlertContacts = () => api.get<AlertContact[]>('/alert-contacts');

export const getAlertContact = (id: string) =>
  api.get<AlertContact>(`/alert-contacts/${id}`);

export const createAlertContact = (data: Record<string, unknown>) =>
  api.post<AlertContact>('/alert-contacts', data);

export const updateAlertContact = (id: string, data: Record<string, unknown>) =>
  api.patch<AlertContact>(`/alert-contacts/${id}`, data);

export const deleteAlertContact = (id: string) =>
  api.delete(`/alert-contacts/${id}`);

export const enableAlertContact = (id: string) =>
  api.post(`/alert-contacts/${id}/enable`);

export const disableAlertContact = (id: string) =>
  api.post(`/alert-contacts/${id}/disable`);
