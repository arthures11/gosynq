import { Component } from '@angular/core';
import { ApiService } from '../../services/api.service';
import { Router } from '@angular/router';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { TitleCasePipe } from '@angular/common';

@Component({
  selector: 'app-create-job',
  templateUrl: './create-job.component.html',
  styleUrls: ['./create-job.component.scss'],
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    TitleCasePipe
  ]
})
export class CreateJobComponent {
  jobData = {
    queue: 'default',
    payload: '{}',
    max_retries: 3,
    priority: 'normal',
    idempotency_key: ''
  };

  error: string | null = null;
  success: string | null = null;
  isSubmitting: boolean = false;

  constructor(
    private apiService: ApiService,
    private router: Router
  ) { }

  createJob(): void {
    this.isSubmitting = true;
    this.error = null;
    this.success = null;

    try {
      // Parse the payload JSON to validate it
      JSON.parse(this.jobData.payload);

      this.apiService.createJob(this.jobData).subscribe({
        next: (response) => {
          this.success = `Job created successfully! ID: ${response.job_id}`;
          this.isSubmitting = false;

          // Reset form after 2 seconds
          setTimeout(() => {
            this.resetForm();
            this.router.navigate(['/jobs']);
          }, 2000);
        },
        error: (err) => {
          this.error = 'Failed to create job: ' + (err.message || 'Unknown error');
          this.isSubmitting = false;
          console.error('Error creating job:', err);
        }
      });
    } catch (e: unknown) {
      const error = e as Error;
      this.error = 'Invalid JSON payload: ' + error.message;
      this.isSubmitting = false;
    }
  }

  resetForm(): void {
    this.jobData = {
      queue: 'default',
      payload: '{}',
      max_retries: 3,
      priority: 'normal',
      idempotency_key: ''
    };
  }

  getPriorityOptions(): string[] {
    return ['low', 'normal', 'high'];
  }
}
