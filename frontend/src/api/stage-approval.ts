
import { request } from './client';
import type { DomainRecord } from '../types/domain';

export async function listStageApproval(page = 1, pageSize = 20, search = '') {
  return request<DomainRecord[]>(`/approvals?page=${page}&pageSize=${pageSize}&search=${encodeURIComponent(search)}`);
}
// 方案页门禁面板需要拿到全量待复核/已批准审批并按关联方案编码归组，
// 这里显式取一个较大页而不是隐式循环分页。
export async function listAllStageApprovalsForGate() {
  return request<DomainRecord[]>('/approvals?page=1&pageSize=100');
}
export async function createStageApproval(input: Partial<DomainRecord>) {
  return request<DomainRecord>('/approvals', { method: 'POST', body: JSON.stringify(input) });
}
export async function transitionStageApproval(id: number, status: string, expectedVersion: number, reason: string) {
  return request<DomainRecord>(`/approvals/${id}/transition`, {
    method: 'POST', body: JSON.stringify({ status, expectedVersion, reason }),
  });
}
