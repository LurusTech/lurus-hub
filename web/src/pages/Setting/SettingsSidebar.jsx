import React from 'react';
import { Select } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import {
  Settings,
  Paintbrush,
  FileText,
  LayoutGrid,
  KeyRound,
  ShieldCheck,
  DollarSign,
  Cpu,
  Sparkles,
  Gauge,
  Activity,
} from 'lucide-react';

const GROUPS = [
  {
    labelKey: 'setting.group.general',
    items: [
      { key: 'general', labelKey: 'setting.nav.general', icon: Settings },
      { key: 'branding', labelKey: 'setting.nav.branding', icon: Paintbrush },
      { key: 'content', labelKey: 'setting.nav.content', icon: FileText },
      { key: 'ui-modules', labelKey: 'setting.nav.ui-modules', icon: LayoutGrid },
    ],
  },
  {
    labelKey: 'setting.group.auth',
    items: [
      { key: 'auth', labelKey: 'setting.nav.auth', icon: KeyRound },
      { key: 'security', labelKey: 'setting.nav.security', icon: ShieldCheck },
    ],
  },
  {
    labelKey: 'setting.group.model',
    items: [
      { key: 'pricing', labelKey: 'setting.nav.pricing', icon: DollarSign },
      { key: 'model-config', labelKey: 'setting.nav.model-config', icon: Cpu },
      { key: 'ai-features', labelKey: 'setting.nav.ai-features', icon: Sparkles },
    ],
  },
  {
    labelKey: 'setting.group.ops',
    items: [
      { key: 'quota-limits', labelKey: 'setting.nav.quota-limits', icon: Gauge },
      { key: 'monitoring', labelKey: 'setting.nav.monitoring', icon: Activity },
    ],
  },
];

export { GROUPS };

const SettingsSidebar = ({ activeKey, onChange }) => {
  const { t } = useTranslation();

  return (
    <>
      {/* Desktop sidebar */}
      <nav className="hidden md:block w-[220px] shrink-0 sticky top-4 self-start">
        <div className="flex flex-col gap-1">
          {GROUPS.map((group) => (
            <div key={group.labelKey} className="mb-2">
              <div className="px-3 py-1 text-xs font-semibold uppercase tracking-wider text-[var(--semi-color-text-2)]">
                {t(group.labelKey)}
              </div>
              {group.items.map((item) => {
                const Icon = item.icon;
                const isActive = activeKey === item.key;
                return (
                  <button
                    key={item.key}
                    onClick={() => onChange(item.key)}
                    className={`w-full flex items-center gap-2 px-3 py-2 rounded-md text-sm cursor-pointer transition-colors
                      ${
                        isActive
                          ? 'bg-[var(--semi-color-primary-light-default)] text-[var(--semi-color-primary)] font-medium border-l-[3px] border-[var(--semi-color-primary)]'
                          : 'text-[var(--semi-color-text-0)] hover:bg-[var(--semi-color-fill-0)] border-l-[3px] border-transparent'
                      }`}
                  >
                    <Icon size={16} />
                    <span>{t(item.labelKey)}</span>
                  </button>
                );
              })}
            </div>
          ))}
        </div>
      </nav>

      {/* Mobile select */}
      <div className="block md:hidden mb-4 w-full">
        <Select
          value={activeKey}
          onChange={onChange}
          style={{ width: '100%' }}
          optionList={GROUPS.flatMap((group) =>
            group.items.map((item) => ({
              label: `${t(group.labelKey)} / ${t(item.labelKey)}`,
              value: item.key,
            })),
          )}
        />
      </div>
    </>
  );
};

export default SettingsSidebar;
