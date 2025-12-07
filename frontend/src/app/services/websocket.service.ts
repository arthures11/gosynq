import { Injectable } from '@angular/core';
import { Observable, Subject } from 'rxjs';
import { webSocket, WebSocketSubject } from 'rxjs/webSocket';

export interface JobEvent {
  type: string;
  job_id: string;
  queue: string;
  timestamp: string;
  payload?: any;
  error?: string;
}

@Injectable({
  providedIn: 'root'
})
export class WebsocketService {
  private socket$: WebSocketSubject<JobEvent> | null = null;
  private messagesSubject = new Subject<JobEvent>();
  public messages$ = this.messagesSubject.asObservable();

  constructor() {
    this.connect();
  }

  private connect(): void {
    // Connect to WebSocket server - adjust URL based on your backend
    const wsUrl = 'ws://localhost:8080/api/v1/ws';
    this.socket$ = webSocket(wsUrl);

    this.socket$.subscribe(
      (message: JobEvent) => {
        console.log('WebSocket message received:', message);
        this.messagesSubject.next(message);
      },
      (error) => {
        console.error('WebSocket error:', error);
        this.reconnect();
      },
      () => {
        console.log('WebSocket connection closed');
        this.reconnect();
      }
    );
  }

  private reconnect(): void {
    console.log('Attempting to reconnect...');
    setTimeout(() => {
      this.connect();
    }, 3000); // Try to reconnect after 3 seconds
  }

  public sendMessage(message: any): void {
    if (this.socket$ && !this.socket$.closed) {
      this.socket$.next(message);
    } else {
      console.error('WebSocket is not connected');
    }
  }

  public close(): void {
    if (this.socket$) {
      this.socket$.complete();
    }
  }
}
