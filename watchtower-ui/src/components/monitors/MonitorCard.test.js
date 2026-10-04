import { useState } from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import MonitorCard from './MonitorCard';

const monitor = { id: 'api', label: 'Public API', endpoint: 'https://example.com', status: 'up', is_enabled: true, probe_interval: 60, network_config: { protocol: 'HTTP', method: 'GET' } };

function Workspace() {
  const [enabled, setEnabled] = useState(true);
  const [opened, setOpened] = useState(false);
  return <><MonitorCard monitor={{ ...monitor, is_enabled: enabled }} onToggle={(id, next) => { if (id === 'api') setEnabled(next); }} onClick={() => setOpened(true)} />{opened && <p>Monitor details opened</p>}</>;
}

test('keyboard users can open a monitor without toggling it', () => {
  render(<Workspace />);
  userEvent.tab();
  expect(screen.getByRole('button', { name: /Public API/ })).toHaveFocus();
  userEvent.keyboard('{Enter}');
  expect(screen.getByText('Monitor details opened')).toBeInTheDocument();
  expect(screen.getByRole('checkbox', { name: /monitoring for Public API/ })).toBeChecked();
});

test('keyboard users can toggle monitoring without opening the card', () => {
  render(<Workspace />);
  userEvent.tab();
  userEvent.tab();
  userEvent.keyboard(' ');
  expect(screen.getByRole('checkbox', { name: /monitoring for Public API/ })).not.toBeChecked();
  expect(screen.queryByText('Monitor details opened')).not.toBeInTheDocument();
});
