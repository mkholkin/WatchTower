import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { AuthProvider, useAuth } from '../context/AuthContext';
import LoginPage from './LoginPage';
import RegisterPage from './RegisterPage';
import * as authApi from '../api/auth';

jest.mock('react-router-dom', () => ({
  Link: ({ to, children, ...props }) => <a href={to} {...props}>{children}</a>,
  useNavigate: () => jest.fn(),
}), { virtual: true });
jest.mock('../api/auth', () => ({
  authError: err => ({401: 'Incorrect code', 410: 'Code expired'})[err.response?.status] || 'Request failed',
  mfaEnabled: true,
  login: jest.fn(), register: jest.fn(), verify: jest.fn(), changePassword: jest.fn(),
}));
function Session() {
  const { user } = useAuth();
  return <div>{user ? `Session: ${user.login}` : 'No session'}</div>;
}
function loginForm() {
  render(<AuthProvider><LoginPage /><Session /></AuthProvider>);
  fireEvent.change(screen.getByLabelText('Username'), { target: { value: 'alice' } });
  fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'fixture-password' } });
  fireEvent.submit(screen.getByLabelText('Password').closest('form'));
}
beforeEach(() => { localStorage.clear(); jest.clearAllMocks(); });

test('password only opens the code form; a session is created only after verification', async () => {
  authApi.login.mockResolvedValue({ data: { challenge_id: 'challenge' } });
  authApi.verify.mockResolvedValue({ data: { access_token: 'verified-token', password_change_required: false } });
  loginForm();
  const code = await screen.findByLabelText('Verification code');
  expect(screen.getByText('No session')).toBeInTheDocument();
  expect(localStorage.getItem('access_token')).toBeNull();
  expect(screen.queryByLabelText('Password')).not.toBeInTheDocument();
  fireEvent.change(code, { target: { value: '123456' } });
  fireEvent.submit(code.closest('form'));
  await screen.findByText('Session: alice');
  expect(localStorage.getItem('access_token')).toBe('verified-token');
});

test('incorrect and expired codes leave the code form open without a session', async () => {
  authApi.login.mockResolvedValue({ data: { challenge_id: 'challenge' } });
  authApi.verify.mockRejectedValueOnce({ response: { status: 401 } })
    .mockRejectedValueOnce({ response: { status: 410 } });
  loginForm();
  const code = await screen.findByLabelText('Verification code');
  fireEvent.change(code, { target: { value: '123456' } });
  fireEvent.submit(code.closest('form'));
  expect(await screen.findByRole('alert')).toHaveTextContent(/incorrect/i);
  fireEvent.submit(code.closest('form'));
  await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent(/expired/i));
  expect(screen.getByLabelText('Verification code')).toBeInTheDocument();
  expect(localStorage.getItem('access_token')).toBeNull();
});

test('incorrect credentials leave recovery available', async () => {
  authApi.login.mockRejectedValue({ response: { status: 401 } });
  loginForm();
  expect(await screen.findByRole('alert')).toHaveTextContent(/incorrect/i);
  expect(screen.getByRole('link', { name: /recover/i })).toHaveAttribute('href', '/recover');
});

test('requesting a new code returns to the password form without retaining credentials', async () => {
  authApi.login.mockResolvedValue({ data: { challenge_id: 'challenge' } });
  loginForm();
  await screen.findByLabelText('Verification code');
  userEvent.click(screen.getByRole('button', { name: /start again/i }));
  expect(screen.getByLabelText('Password')).toHaveValue('');
  expect(screen.queryByLabelText('Verification code')).not.toBeInTheDocument();
});

test('password-change-only token is never stored as a full session', async () => {
  authApi.login.mockResolvedValue({ data: { challenge_id: 'challenge' } });
  authApi.verify.mockResolvedValue({ data: { access_token: 'restricted-token', password_change_required: true } });
  loginForm();
  const code = await screen.findByLabelText('Verification code');
  fireEvent.change(code, { target: { value: '123456' } });
  fireEvent.submit(code.closest('form'));
  await waitFor(() => expect(authApi.verify).toHaveBeenCalled());
  expect(screen.getByText('No session')).toBeInTheDocument();
  expect(localStorage.getItem('access_token')).toBeNull();
});

test('MFA registration collects email and does not automatically create a session', async () => {
  authApi.register.mockResolvedValue({ data: { login: 'alice' } });
  render(<AuthProvider><RegisterPage /><Session /></AuthProvider>);
  fireEvent.change(screen.getByLabelText('Username'), { target: { value: 'alice' } });
  fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'alice@example.test' } });
  fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'fixture-password' } });
  fireEvent.submit(screen.getByLabelText('Password').closest('form'));
  await waitFor(() => expect(authApi.register).toHaveBeenCalledWith('alice', 'fixture-password', 'alice@example.test'));
  expect(screen.getByText('No session')).toBeInTheDocument();
  expect(localStorage.getItem('access_token')).toBeNull();
});
