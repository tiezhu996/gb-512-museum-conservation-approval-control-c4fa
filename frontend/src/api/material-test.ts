
import { request } from './client';
import type { DomainRecord } from '../types/domain';

export async function listMaterialTest(page = 1, pageSize = 20, search = '') {
  return request<DomainRecord[]>(`/tests?page=${page}&pageSize=${pageSize}&search=${encodeURIComponent(search)}`);
}
export async function createMaterialTest(input: Partial<DomainRecord>) {
  return request<DomainRecord>('/tests', { method: 'POST', body: JSON.stringify(input) });
}
export async function transitionMaterialTest(id: number, status: string, expectedVersion: number, reason: string) {
  return request<DomainRecord>(`/tests/${id}/transition`, {
    method: 'POST', body: JSON.stringify({ status, expectedVersion, reason }),
  });
}
