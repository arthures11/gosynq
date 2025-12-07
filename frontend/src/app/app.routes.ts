import { Routes } from '@angular/router';
import { DashboardComponent } from './components/dashboard/dashboard.component';
import { JobsComponent } from './components/jobs/jobs.component';
import { CreateJobComponent } from './components/create-job/create-job.component';
import { MetricsComponent } from './components/metrics/metrics.component';

export const routes: Routes = [
  {
    path: '',
    component: DashboardComponent,
    children: [
      { path: 'jobs', component: JobsComponent },
      { path: 'create', component: CreateJobComponent },
      { path: 'metrics', component: MetricsComponent },
      { path: '', redirectTo: 'jobs', pathMatch: 'full' }
    ]
  },
  { path: '**', redirectTo: '' }
];
