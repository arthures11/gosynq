import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';

export interface Job {
  id: string;
  queue: string;
  payload: any;
  max_retries: number;
  run_at: string;
  created_at: string;
  updated_at: string;
  status: string;
  priority: string;
  idempotency_key: string;
  locked_by: string;
  locked_at?: string;
}

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  private apiUrl = environment.apiUrl || 'http://localhost:8080/api/v1';

  constructor(private http: HttpClient) { }

  // Get all jobs
  getJobs(status?: string, queue?: string, limit: number = 100): Observable<Job[]> {
    let url = `${this.apiUrl}/jobs`;
    const params: any = { limit: limit.toString() };

    if (status) params.status = status;
    if (queue) params.queue = queue;

    return this.http.get<Job[]>(url, { params });
  }

  // Get job by ID
  getJobById(jobId: string): Observable<Job> {
    return this.http.get<Job>(`${this.apiUrl}/jobs/${jobId}`);
  }

  // Create a new job
  createJob(jobData: {
    queue: string;
    payload: any;
    max_retries: number;
    priority: string;
    idempotency_key?: string;
  }): Observable<{ job_id: string; status: string }> {
    return this.http.post<{ job_id: string; status: string }>(`${this.apiUrl}/jobs`, jobData);
  }

  // Cancel a job (admin)
  cancelJob(jobId: string): Observable<{ status: string }> {
    const headers = new HttpHeaders({
      'Authorization': 'Basic ' + btoa('admin:password')
    });
    return this.http.post<{ status: string }>(`${this.apiUrl}/admin/jobs/${jobId}/cancel`, {}, { headers });
  }

  // Retry a job (admin)
  retryJob(jobId: string): Observable<{ status: string }> {
    const headers = new HttpHeaders({
      'Authorization': 'Basic ' + btoa('admin:password')
    });
    return this.http.post<{ status: string }>(`${this.apiUrl}/admin/jobs/${jobId}/retry`, {}, { headers });
  }

  // Get health status
  getHealth(): Observable<{ status: string }> {
    return this.http.get<{ status: string }>(`${this.apiUrl}/health`);
  }

  // Get job statistics
  getJobStats(): Observable<{
    total_jobs: number;
    pending_jobs: number;
    processing_jobs: number;
    completed_jobs: number;
    failed_jobs: number;
    cancelled_jobs: number;
  }> {
    return this.http.get<{
      total_jobs: number;
      pending_jobs: number;
      processing_jobs: number;
      completed_jobs: number;
      failed_jobs: number;
      cancelled_jobs: number;
    }>(`${this.apiUrl}/stats`);
  }
}
