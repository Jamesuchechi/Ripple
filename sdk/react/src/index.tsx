import React, { createContext, useContext, useEffect, useState } from 'react';
import { RippleClient, Activity } from '@ripple/js';

interface RippleContextType {
  client: RippleClient | null;
}

const RippleContext = createContext<RippleContextType>({ client: null });

export const RippleProvider: React.FC<{ client: RippleClient; children: React.ReactNode }> = ({
  client,
  children,
}) => {
  return <RippleContext.Provider value={{ client }}>{children}</RippleContext.Provider>;
};

export const useRipple = (): RippleClient => {
  const ctx = useContext(RippleContext);
  if (!ctx.client) {
    throw new Error('useRipple must be used within a <RippleProvider>');
  }
  return ctx.client;
};

export const useUnreadCount = (userID: string): number => {
  const client = useRipple();
  const [count, setCount] = useState<number>(0);

  useEffect(() => {
    client.getUnreadCount(userID).then((res) => setCount(res.total_unread)).catch(console.error);

    client.connectWebSocket(userID);
    const unsubscribe = client.onUnreadCount((newCount) => setCount(newCount));
    return () => unsubscribe();
  }, [client, userID]);

  return count;
};

export const useFeed = (userID: string, limit = 20): { activities: Activity[]; loading: boolean; refresh: () => void } => {
  const client = useRipple();
  const [activities, setActivities] = useState<Activity[]>([]);
  const [loading, setLoading] = useState<boolean>(true);

  const fetchFeed = () => {
    setLoading(true);
    client.getFeed(userID, limit)
      .then((res) => setActivities(res.activities))
      .catch(console.error)
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchFeed();
    const unsubscribe = client.onNotification((newAct) => {
      setActivities((prev) => [newAct, ...prev]);
    });
    return () => unsubscribe();
  }, [client, userID, limit]);

  return { activities, loading, refresh: fetchFeed };
};

export const NotificationBadge: React.FC<{ userID: string }> = ({ userID }) => {
  const count = useUnreadCount(userID);
  if (count === 0) return null;

  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '0.15rem 0.45rem',
        borderRadius: '9999px',
        backgroundColor: '#f43f5e',
        color: '#ffffff',
        fontSize: '0.75rem',
        fontWeight: 'bold',
      }}
    >
      {count > 99 ? '99+' : count}
    </span>
  );
};

export const NotificationFeedList: React.FC<{ userID: string }> = ({ userID }) => {
  const { activities, loading } = useFeed(userID);

  if (loading) return <div>Loading notifications...</div>;
  if (activities.length === 0) return <div>No notifications yet.</div>;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
      {activities.map((act) => (
        <div
          key={act.event_id}
          style={{
            padding: '0.75rem',
            borderRadius: '8px',
            background: 'rgba(255, 255, 255, 0.05)',
            border: '1px solid rgba(255, 255, 255, 0.1)',
          }}
        >
          <strong>{act.actor_id}</strong> {act.verb} object <code>{act.object_id}</code>
          <div style={{ fontSize: '0.75rem', color: '#9ca3af', marginTop: '0.2rem' }}>
            {act.created_at}
          </div>
        </div>
      ))}
    </div>
  );
};
