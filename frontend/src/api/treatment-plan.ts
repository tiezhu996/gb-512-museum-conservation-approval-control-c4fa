
import { request } from './client';
import type { DomainRecord } from '../types/domain';

export async function listTreatmentPlan(page = 1, pageSize = 20, search = '') {
  return request<DomainRecord[]>(`/plans?page=${page}&pageSize=${pageSize}&search=${encodeURIComponent(search)}`);
}
export async function createTreatmentPlan(input: Partial<DomainRecord>) {
  return request<DomainRecord>('/plans', { method: 'POST', body: JSON.stringify(input) });
}
export async function transitionTreatmentPlan(id: number, status: string, expectedVersion: number, reason: string) {
  return request<DomainRecord>(`/plans/${id}/transition`, {
    method: 'POST', body: JSON.stringify({ status, expectedVersion, reason }),
  });
}
