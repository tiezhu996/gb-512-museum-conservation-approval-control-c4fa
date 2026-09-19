
export function formatDate(value: string): string {
  return value ? new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : '-';
}
export function nextStatus(current: string, statuses: readonly string[]): string | null {
  const index = statuses.indexOf(current);
  return index >= 0 && index < statuses.length - 1 ? statuses[index + 1] : null;
}
export function statusTone(status: string): 'success' | 'warning' | 'danger' | 'neutral' {
  if (/approved|accepted|released|completed|signed|closed|pass|ready|online|cleared|succeeded/.test(status)) return 'success';
  if (/failed|rejected|critical|scrap|discard|revoked|urgent/.test(status)) return 'danger';
  if (/hold|warning|review|pending|restricted|limited|quarantine/.test(status)) return 'warning';
  return 'neutral';
}

// gateVerdictTone maps the persisted approval-gate conclusion onto the shared
// tone scale so the approvals table and the plan page panel render it alike.
export function gateVerdictTone(verdict?: string): 'success' | 'warning' | 'danger' | 'neutral' {
  if (!verdict) return 'neutral';
  if (verdict.includes('拦截')) return 'danger';
  if (verdict.includes('通过')) return 'success';
  if (verdict.includes('冻结')) return 'warning';
  return 'neutral';
}
