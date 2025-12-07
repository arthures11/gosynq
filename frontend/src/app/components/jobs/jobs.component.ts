import { Component, OnInit, OnDestroy } from '@angular/core';
import { ApiService, Job } from '../../services/api.service';
import { WebsocketService, JobEvent } from '../../services/websocket.service';
import { Subscription } from 'rxjs';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { DatePipe, SlicePipe } from '@angular/common';

@Component({
  selector: 'app-jobs',
  templateUrl: './jobs.component.html',
  styleUrls: ['./jobs.component.scss'],
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    DatePipe,
    SlicePipe
  ]
})
export class JobsComponent implements OnInit, OnDestroy {
  jobs: Job[] = [];
  filteredJobs: Job[] = [];
  statusFilter: string = '';
  queueFilter: string = '';
  isLoading: boolean = false;
  error: string | null = null;

  private wsSubscription: Subscription;
  jobEvents: JobEvent[] = [];

  constructor(
    private apiService: ApiService,
    private websocketService: WebsocketService
  ) {
    this.wsSubscription = this.websocketService.messages$.subscribe((event: JobEvent) => {
      this.handleJobEvent(event);
    });
  }

  ngOnInit(): void {
    this.loadJobs();
  }

  ngOnDestroy(): void {
    if (this.wsSubscription) {
      this.wsSubscription.unsubscribe();
    }
  }

  loadJobs(): void {
    this.isLoading = true;
    this.error = null;

    this.apiService.getJobs(this.statusFilter, this.queueFilter)
      .subscribe({
        next: (jobs) => {
          this.jobs = jobs;
          this.filteredJobs = [...jobs];
          this.isLoading = false;
        },
        error: (err) => {
          this.error = 'Failed to load jobs: ' + (err.message || 'Unknown error');
          this.isLoading = false;
          console.error('Error loading jobs:', err);
        }
      });
  }

  handleJobEvent(event: JobEvent): void {
    console.log('Job event received:', event);

    // Add event to our events list
    this.jobEvents.unshift(event); // Add to beginning to show newest first

    // Update the specific job in our list
    const jobIndex = this.jobs.findIndex(j => j.id === event.job_id);
    if (jobIndex !== -1) {
      // Update job status
      this.jobs[jobIndex].status = event.type;
      this.filteredJobs = [...this.jobs]; // Refresh filtered list
    } else {
      // If job doesn't exist in our list, we might need to reload
      this.loadJobs();
    }
  }

  applyFilters(): void {
    this.filteredJobs = this.jobs.filter(job => {
      const statusMatch = !this.statusFilter || job.status === this.statusFilter;
      const queueMatch = !this.queueFilter || job.queue === this.queueFilter;
      return statusMatch && queueMatch;
    });
  }

  cancelJob(jobId: string): void {
    if (confirm('Are you sure you want to cancel this job?')) {
      this.apiService.cancelJob(jobId).subscribe({
        next: () => {
          this.loadJobs(); // Refresh the list
        },
        error: (err) => {
          this.error = 'Failed to cancel job: ' + (err.message || 'Unknown error');
          console.error('Error canceling job:', err);
        }
      });
    }
  }

  retryJob(jobId: string): void {
    if (confirm('Are you sure you want to retry this job?')) {
      this.apiService.retryJob(jobId).subscribe({
        next: () => {
          this.loadJobs(); // Refresh the list
        },
        error: (err) => {
          this.error = 'Failed to retry job: ' + (err.message || 'Unknown error');
          console.error('Error retrying job:', err);
        }
      });
    }
  }

  getStatusColor(status: string): string {
    switch (status) {
      case 'completed': return 'bg-green-100 text-green-800';
      case 'failed': return 'bg-red-100 text-red-800';
      case 'cancelled': return 'bg-gray-100 text-gray-800';
      case 'processing': return 'bg-blue-100 text-blue-800';
      case 'pending': return 'bg-yellow-100 text-yellow-800';
      default: return 'bg-purple-100 text-purple-800';
    }
  }

  getStatusBadge(status: string): string {
    switch (status) {
      case 'completed': return '✅ Completed';
      case 'failed': return '❌ Failed';
      case 'cancelled': return '🚫 Cancelled';
      case 'processing': return '🔄 Processing';
      case 'pending': return '⏳ Pending';
      default: return status;
    }
  }
}
