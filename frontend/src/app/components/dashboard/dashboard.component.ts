import { Component, OnInit } from '@angular/core';
import { ApiService } from '../../services/api.service';
import { CommonModule } from '@angular/common';
import { DatePipe } from '@angular/common';
import { RouterModule } from '@angular/router';

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.scss'],
  standalone: true,
  imports: [
    CommonModule,
    DatePipe,
    RouterModule
  ]
})
export class DashboardComponent implements OnInit {
  healthStatus: string = 'loading';
  serverInfo: any = null;
  stats: any = {
    totalJobs: 0,
    pendingJobs: 0,
    processingJobs: 0,
    completedJobs: 0,
    failedJobs: 0
  };

  constructor(private apiService: ApiService) { }

  ngOnInit(): void {
    this.checkHealth();
    this.loadStats();
  }

  checkHealth(): void {
    this.apiService.getHealth().subscribe({
      next: (response) => {
        this.healthStatus = response.status;
        this.serverInfo = {
          status: response.status,
          timestamp: new Date().toISOString()
        };
      },
      error: (err) => {
        this.healthStatus = 'unhealthy';
        console.error('Health check failed:', err);
      }
    });
  }

  loadStats(): void {
    this.apiService.getJobStats().subscribe({
      next: (stats) => {
        this.stats = {
          totalJobs: stats.total_jobs || 0,
          pendingJobs: stats.pending_jobs || 0,
          processingJobs: stats.processing_jobs || 0,
          completedJobs: stats.completed_jobs || 0,
          failedJobs: stats.failed_jobs || 0
        };
      },
      error: (err) => {
        console.error('Failed to load job stats:', err);
      }
    });
  }

  getHealthColor(): string {
    switch (this.healthStatus) {
      case 'healthy': return 'bg-green-100 text-green-800';
      case 'unhealthy': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  }
}
