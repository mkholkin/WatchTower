export type MonitorStatus = 'up' | 'down' | 'maintenance' | 'unknown';
export type MonitorProtocol = 'HTTP';
export type HTTPMethod = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH' | 'HEAD' | 'OPTIONS';
export type MaintenanceWindowType = 'one_time' | 'manual';
export type AlertContactPlatform = 'telegram';

export interface HTTPConfig {
  protocol: 'HTTP';
  method: HTTPMethod;
  headers: Record<string, string> | null;
  body: string | null;
  follow_redirects: boolean;
}

export interface HTTPExpectations {
  protocol: 'HTTP';
  expected_status_codes: number[];
  expected_response_time_ms: number;
}

export interface TelegramConfig {
  platform: 'telegram';
  chat_id: number;
  token: string;
}

export interface AlertContact {
  id: string;
  name: string;
  config: TelegramConfig;
  is_enabled: boolean;
}

export interface OneTimeConfig {
  type: 'one_time';
  start_time: string;
  end_time: string;
}

export interface ManualConfig {
  type: 'manual';
  is_active: boolean;
}

export type MaintenanceWindowConfig = OneTimeConfig | ManualConfig;

export interface MaintenanceWindow {
  id: string;
  title: string;
  description: string | null;
  config: MaintenanceWindowConfig;
}

export interface Monitor {
  id: string;
  label: string;
  description: string | null;
  endpoint: string;
  probe_interval: number;
  network_config: HTTPConfig;
  expectations: HTTPExpectations;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
  status: MonitorStatus;
  alert_contacts: AlertContact[];
  maintenance_windows: MaintenanceWindow[];
}

export interface MonitorCheck {
  monitor_id: string;
  check_time: string;
  status: MonitorStatus;
  latency_ms: number;
  status_code: number | null;
  network_failure: boolean;
  failure_reason: string | null;
}

export interface MonitorStatusEvent {
  monitor_id: string;
  status: MonitorStatus;
  start_time: string;
  end_time: string | null;
  reason: string | null;
}

export interface MonitorSLA {
  monitor_id: string;
  start_time: string;
  end_time: string;
  uptime_percentage: number;
  downtime_duration_sec: number;
}

export interface ErrorResponse {
  code: string;
  message: string;
}

export interface SuccessResponse {
  success: boolean;
  message: string;
}
