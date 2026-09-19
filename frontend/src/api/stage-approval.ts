
import { request } from './client';
import type { DomainRecord } from '../types/domain';

export async function listStageApproval(page = 1, pageSize = 20, search = '') {
  return request<DomainRecord[]>(`/approvals?page=${page}&pageSize=${pageSize}&search=${encodeURIComponent(search)}`);
}
export async function createStageApproval(input: Partial<DomainRecord>) {
  return request<DomainRecord>('/approvals', { method: 'POST', body: JSON.stringify(input) });
}
export async function transitionStageApproval(id: number, status: string, expectedVersion: number, reason: string) {
  return request<DomainRecord>(`/approvals/${id}/transition`, {
    method: 'POST', body: JSON.stringify({ status, expectedVersion, reason }),
  });
}
