export interface RippleOptions {
  apiKey: string;
  baseUrl?: string;
  wsUrl?: string;
}

export interface Activity {
  event_id: string;
  project_id: string;
  trace_id?: string;
  verb: string;
  actor_id: string;
  object_id: string;
  target_id?: string;
  recipients?: string[];
  payload?: any;
  created_at: string;
}

export interface WSMessage {
  type: string;
  activity?: Activity;
  unread_count?: number;
  timestamp: string;
}

export interface FeedResponse {
  activities: Activity[];
  next_cursor?: string;
}

export interface UnreadCountResponse {
  user_id: string;
  total_unread: number;
  by_channel: Record<string, number>;
}

export class RippleClient {
  private apiKey: string;
  private baseUrl: string;
  private wsUrl: string;
  private ws: WebSocket | null = null;
  private notificationListeners: Array<(activity: Activity) => void> = [];
  private unreadCountListeners: Array<(count: number) => void> = [];
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;

  constructor(options: RippleOptions) {
    this.apiKey = options.apiKey;
    this.baseUrl = options.baseUrl || 'http://localhost:8080';
    this.wsUrl = options.wsUrl || 'ws://localhost:8081';
  }

  // Connects real-time WebSocket stream for a recipient
  public connectWebSocket(userID: string): void {
    const url = `${this.wsUrl}/v1/ws?user_id=${encodeURIComponent(userID)}`;
    this.ws = new WebSocket(url);

    this.ws.onopen = () => {
      console.log(`[Ripple SDK] WebSocket connected for user ${userID}`);
      this.reconnectAttempts = 0;
    };

    this.ws.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data);
        if (msg.type === 'notification' && msg.activity) {
          this.notificationListeners.forEach((cb) => cb(msg.activity!));
        }
        if (msg.unread_count !== undefined) {
          this.unreadCountListeners.forEach((cb) => cb(msg.unread_count!));
        }
      } catch (err) {
        console.error('[Ripple SDK] Error parsing WS frame:', err);
      }
    };

    this.ws.onclose = () => {
      console.warn('[Ripple SDK] WebSocket connection closed.');
      this.attemptReconnect(userID);
    };
  }

  private attemptReconnect(userID: string): void {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      const delay = Math.pow(2, this.reconnectAttempts) * 1000;
      setTimeout(() => this.connectWebSocket(userID), delay);
    }
  }

  public onNotification(callback: (activity: Activity) => void): () => void {
    this.notificationListeners.push(callback);
    return () => {
      this.notificationListeners = this.notificationListeners.filter((cb) => cb !== callback);
    };
  }

  public onUnreadCount(callback: (count: number) => void): () => void {
    this.unreadCountListeners.push(callback);
    return () => {
      this.unreadCountListeners = this.unreadCountListeners.filter((cb) => cb !== callback);
    };
  }

  // REST API Methods
  public async getFeed(userID: string, limit = 20, cursor?: string): Promise<FeedResponse> {
    const params = new URLSearchParams({ limit: limit.toString() });
    if (cursor) params.append('cursor', cursor);

    const res = await fetch(`${this.baseUrl}/v1/feed/${userID}?${params.toString()}`, {
      headers: {
        Authorization: `Bearer ${this.apiKey}`,
      },
    });
    if (!res.ok) throw new Error(`Ripple API error: ${res.statusText}`);
    return res.json();
  }

  public async getUnreadCount(userID: string): Promise<UnreadCountResponse> {
    const res = await fetch(`${this.baseUrl}/v1/notifications/${userID}/unread_count`, {
      headers: {
        Authorization: `Bearer ${this.apiKey}`,
      },
    });
    if (!res.ok) throw new Error(`Ripple API error: ${res.statusText}`);
    return res.json();
  }

  public async markRead(userID: string): Promise<void> {
    const res = await fetch(`${this.baseUrl}/v1/notifications/mark_read`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${this.apiKey}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ user_id: userID }),
    });
    if (!res.ok) throw new Error(`Ripple API error: ${res.statusText}`);
  }

  public async ingestEvent(event: {
    verb: string;
    actor_id: string;
    object_id: string;
    target_id?: string;
    recipients?: string[];
    payload?: any;
    dedup_key?: string;
  }): Promise<{ event_id: string; status: string }> {
    const res = await fetch(`${this.baseUrl}/v1/events`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${this.apiKey}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(event),
    });
    if (!res.ok) throw new Error(`Ripple API error: ${res.statusText}`);
    return res.json();
  }

  public disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}
