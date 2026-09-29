import { ReactNode } from 'react';
import { Activity, Bell, Clock3, Radio } from 'lucide-react';

export default function AuthLayout({ title, subtitle, children }: { title: string; subtitle: string; children: ReactNode }) {
  return (
    <main className="min-h-screen bg-app-bg lg:grid lg:grid-cols-2">
      <section className="auth-grid relative flex flex-col justify-between border-b border-border bg-sidebar-bg p-8 sm:p-12 lg:min-h-screen lg:border-b-0 lg:border-r lg:p-16">
        <div className="flex items-center gap-3 text-xl font-semibold tracking-tight text-slate-100">
          <span className="flex h-10 w-10 items-center justify-center rounded-xl border border-emerald-400/20 bg-emerald-400/10 text-emerald-300"><Activity size={23} /></span>
          <span>WatchTower<span className="text-emerald-400">.</span></span>
        </div>
        <div className="hidden max-w-lg py-20 lg:block">
          <p className="eyebrow mb-6 text-emerald-300">A clearer view of your services</p>
          <h2 className="text-4xl font-semibold leading-[1.12] tracking-tight text-slate-100 sm:text-5xl xl:text-6xl">Keep watch.<br /><span className="text-emerald-300">Stay ahead.</span></h2>
          <p className="mt-6 max-w-sm text-base leading-relaxed text-slate-400">Know when your services need you. Monitor availability, follow performance, and bring every alert into focus.</p>
          <div className="mt-10 hidden space-y-5 sm:block">
            {[{ icon: Radio, text: 'Availability at a glance' }, { icon: Bell, text: 'Alerts where you need them' }, { icon: Clock3, text: 'Planned downtime, handled' }].map(({ icon: Icon, text }) => (
              <div key={text} className="flex items-center gap-3 text-sm text-slate-300"><Icon size={17} className="text-emerald-300" />{text}</div>
            ))}
          </div>
        </div>
        <p className="hidden text-xs text-slate-500 lg:block">Self-hosted monitoring. Your infrastructure, your view.</p>
      </section>
      <section className="flex items-center justify-center px-6 py-12 sm:px-12">
        <div className="w-full max-w-sm">
          <p className="eyebrow mb-4">Your monitoring workspace</p>
          <h1 className="text-3xl font-semibold tracking-tight text-slate-100">{title}</h1>
          <p className="mb-8 mt-3 text-sm leading-relaxed text-slate-400">{subtitle}</p>
          {children}
        </div>
      </section>
    </main>
  );
}
