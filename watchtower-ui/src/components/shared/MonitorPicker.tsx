import { useState, useEffect, useCallback } from 'react';
import { Plus, X, GripVertical } from 'lucide-react';
import { Monitor } from '../../types';
import * as monitorsApi from '../../api/monitors';

interface MonitorPickerProps {
  selectedIds: string[];
  onChange: (ids: string[]) => void;
}

export default function MonitorPicker({ selectedIds, onChange }: MonitorPickerProps) {
  const [allMonitors, setAllMonitors] = useState<Monitor[]>([]);
  const [dragOverAssigned, setDragOverAssigned] = useState(false);
  const [dragOverAvailable, setDragOverAvailable] = useState(false);

  useEffect(() => {
    monitorsApi.getMonitors().then((res) => setAllMonitors(res.data));
  }, []);

  const selected = allMonitors.filter((m) => selectedIds.includes(m.id));
  const available = allMonitors.filter((m) => !selectedIds.includes(m.id));

  const add = useCallback(
    (id: string) => { if (!selectedIds.includes(id)) onChange([...selectedIds, id]); },
    [selectedIds, onChange]
  );
  const remove = useCallback(
    (id: string) => onChange(selectedIds.filter((x) => x !== id)),
    [selectedIds, onChange]
  );

  const handleDragStart = (e: React.DragEvent, id: string, from: 'selected' | 'available') => {
    e.dataTransfer.setData('text/plain', JSON.stringify({ id, from }));
    e.dataTransfer.effectAllowed = 'move';
  };

  const handleDragOver = (e: React.DragEvent, area: 'selected' | 'available') => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    if (area === 'selected') setDragOverAssigned(true);
    else setDragOverAvailable(true);
  };

  const handleDragLeave = (area: 'selected' | 'available') => {
    if (area === 'selected') setDragOverAssigned(false);
    else setDragOverAvailable(false);
  };

  const handleDrop = (e: React.DragEvent, area: 'selected' | 'available') => {
    e.preventDefault();
    setDragOverAssigned(false);
    setDragOverAvailable(false);
    try {
      const { id, from }: { id: string; from: 'selected' | 'available' } = JSON.parse(e.dataTransfer.getData('text/plain'));
      if (from === area) return; // same area, no change
      if (area === 'selected') add(id);
      else remove(id);
    } catch {}
  };

  const MiniCard = ({ monitor, area }: { monitor: Monitor; area: 'selected' | 'available' }) => (
    <div
      draggable
      onDragStart={(e) => handleDragStart(e, monitor.id, area)}
      className={`flex items-center gap-2 py-1.5 px-2 rounded-md text-xs cursor-grab active:cursor-grabbing transition-colors select-none group ${
        area === 'selected' ? 'bg-emerald-500/5 border border-emerald-500/15 hover:bg-emerald-500/10' : 'bg-app-bg border border-border hover:bg-card-hover'
      }`}
    >
      <GripVertical size={11} className="text-slate-600 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity" />
      <span className="flex-1 text-slate-300 truncate">{monitor.label}</span>
      <span className="text-slate-600 font-mono text-[10px] shrink-0">{monitor.endpoint.slice(0, 30)}</span>
      {area === 'available' ? (
        <button type="button" onClick={() => add(monitor.id)} className="text-emerald-400 hover:text-emerald-300 shrink-0"><Plus size={13} /></button>
      ) : (
        <button type="button" onClick={() => remove(monitor.id)} className="text-slate-500 hover:text-red-400 shrink-0"><X size={13} /></button>
      )}
    </div>
  );

  const areaClass = (active: boolean) =>
    `border rounded-xl p-3 min-h-[80px] max-h-[220px] overflow-y-auto space-y-1 transition-colors ${
      active ? 'border-emerald-500/40 bg-emerald-500/5' : 'border-border'
    }`;

  return (
    <div className="space-y-4">
      {/* Assigned */}
      <div>
        <p className="text-[11px] font-semibold text-slate-500 uppercase tracking-wider mb-2">
          Assigned ({selected.length})
        </p>
        <div
          className={areaClass(dragOverAssigned)}
          onDragOver={(e) => handleDragOver(e, 'selected')}
          onDragLeave={() => handleDragLeave('selected')}
          onDrop={(e) => handleDrop(e, 'selected')}
        >
          {selected.length === 0 ? (
            <p className="text-xs text-slate-600 text-center py-4">Drag monitors here or click + below</p>
          ) : (
            selected.map((m) => <MiniCard key={m.id} monitor={m} area="selected" />)
          )}
        </div>
      </div>

      {/* Available */}
      <div>
        <p className="text-[11px] font-semibold text-slate-500 uppercase tracking-wider mb-2">
          Available ({available.length})
        </p>
        <div
          className={areaClass(dragOverAvailable)}
          onDragOver={(e) => handleDragOver(e, 'available')}
          onDragLeave={() => handleDragLeave('available')}
          onDrop={(e) => handleDrop(e, 'available')}
        >
          {available.length === 0 ? (
            <p className="text-xs text-slate-600 text-center py-4">No monitors available</p>
          ) : (
            available.map((m) => <MiniCard key={m.id} monitor={m} area="available" />)
          )}
        </div>
      </div>
    </div>
  );
}
