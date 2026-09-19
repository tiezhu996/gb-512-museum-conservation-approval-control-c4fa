
import { request } from './client';
import type { DomainRecord } from '../types/domain';

export async function listArtifact(page = 1, pageSize = 20, search = '') {
  return request<DomainRecord[]>(`/artifacts?page=${page}&pageSize=${pageSize}&search=${encodeURIComponent(search)}`);
}
export async function createArtifact(input: Partial<DomainRecord>) {
  return request<DomainRecord>('/artifacts', { method: 'POST', body: JSON.stringify(input) });
}
export async function transitionArtifact(id: number, status: string, expectedVersion: number, reason: string) {
  return request<DomainRecord>(`/artifacts/${id}/transition`, {
    method: 'POST', body: JSON.stringify({ status, expectedVersion, reason }),
  });
}
