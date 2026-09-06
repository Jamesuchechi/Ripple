import React, { useState } from 'react'

const initialLogs = [
  {
    id: 'dl_88192310',
    eventID: 'evt_9918231',
    verb: 'post.created',
    actorID: 'usr_alice',
    channel: 'webhook',
    status: 'delivered',
    traceID: '4f29a0b12c8841e2a91b2c34001aef88',
    timestamp: '2026-09-06T12:02:14Z',
    latency: '8.4 ms',
  },
  {
    id: 'dl_88192311',
    eventID: 'evt_9918232',
    verb: 'user.followed',
    actorID: 'usr_bob',
    channel: 'push',
    status: 'delivered',
    traceID: '98a72b11e2401f8a721c009b76214152',
    timestamp: '2026-09-06T12:02:10Z',
    latency: '14.1 ms',
  },
  {
    id: 'dl_88192312',
    eventID: 'evt_9918233',
    verb: 'comment.liked',
    actorID: 'usr_charlie',
    channel: 'email',
    status: 'delivered',
    traceID: '12b84f0091cae74092b719941a881920',
    timestamp: '2026-09-06T12:01:45Z',
    latency: '110 ms',
  },
  {
    id: 'dl_88192313',
    eventID: 'evt_9918234',
    verb: 'system.alert',
    actorID: 'usr_sys',
    channel: 'sms',
    status: 'failed',
    traceID: '77c019a2e411b901a740912f883719b2',
    timestamp: '2026-09-06T12:00:12Z',
    latency: '340 ms',
  },
  {
    id: 'dl_88192314',
    eventID: 'evt_9918235',
    verb: 'photo.shared',
    actorID: 'usr_taylor',
    channel: 'in_app',
    status: 'delivered',
    traceID: '88102a9b4412e091b7a60129f12344a0',
    timestamp: '2026-09-06T11:59:58Z',
    latency: '3.1 ms',
  },
]

export default function DeliveryLogsView() {
  const [logs] = useState(initialLogs)
  const [statusFilter, setStatusFilter] = useState('all')
  const [channelFilter, setChannelFilter] = useState('all')
  const [selectedLog, setSelectedLog] = useState(null)

  const filteredLogs = logs.filter((log) => {
    if (statusFilter !== 'all' && log.status !== statusFilter) return false
    if (channelFilter !== 'all' && log.channel !== channelFilter) return false
    return true
  })

  return (
    <div>
      <div className="section-header">
        <div>
          <h1 className="section-title">Live Delivery Log Table</h1>
          <p className="section-desc">
            Real-time feed of multi-channel notification dispatches, status traces, and latencies.
          </p>
        </div>

        <div style={{ display: 'flex', gap: '0.75rem' }}>
          <select
            className="input"
            style={{ width: 'auto' }}
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
          >
            <option value="all">All Statuses</option>
            <option value="delivered">Delivered</option>
            <option value="failed">Failed</option>
            <option value="retrying">Retrying</option>
          </select>

          <select
            className="input"
            style={{ width: 'auto' }}
            value={channelFilter}
            onChange={(e) => setChannelFilter(e.target.value)}
          >
            <option value="all">All Channels</option>
            <option value="webhook">Webhook</option>
            <option value="push">Push (FCM/APNs)</option>
            <option value="email">Email</option>
            <option value="sms">SMS</option>
            <option value="in_app">In-App ZSET</option>
          </select>
        </div>
      </div>

      <div className="glass-card" style={{ padding: 0, overflow: 'hidden' }}>
        <table className="table">
          <thead>
            <tr>
              <th>Log ID</th>
              <th>Event ID</th>
              <th>Verb</th>
              <th>Actor</th>
              <th>Channel</th>
              <th>Status</th>
              <th>Latency</th>
              <th>Timestamp</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            {filteredLogs.map((log) => (
              <tr key={log.id}>
                <td className="code-font">{log.id}</td>
                <td className="code-font">{log.eventID}</td>
                <td>
                  <span className="badge badge-purple">{log.verb}</span>
                </td>
                <td className="code-font">{log.actorID}</td>
                <td>
                  <span className="badge badge-cyan">{log.channel}</span>
                </td>
                <td>
                  <span
                    className={`badge ${
                      log.status === 'delivered'
                        ? 'badge-success'
                        : log.status === 'failed'
                        ? 'badge-error'
                        : 'badge-warning'
                    }`}
                  >
                    {log.status}
                  </span>
                </td>
                <td className="code-font">{log.latency}</td>
                <td style={{ color: 'var(--text-muted)' }}>{log.timestamp}</td>
                <td>
                  <button
                    className="btn btn-secondary"
                    style={{ padding: '0.25rem 0.6rem', fontSize: '0.75rem' }}
                    onClick={() => setSelectedLog(log)}
                  >
                    Inspect
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {selectedLog && (
        <div
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: 'rgba(0,0,0,0.7)',
            backdropFilter: 'blur(8px)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 100,
          }}
        >
          <div className="glass-card" style={{ width: '550px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '1rem' }}>
              <h3>Delivery Inspector — {selectedLog.id}</h3>
              <button
                className="btn btn-secondary"
                style={{ padding: '0.2rem 0.5rem' }}
                onClick={() => setSelectedLog(null)}
              >
                ✕
              </button>
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
              <div>
                <strong style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>Trace ID:</strong>
                <div className="code-font" style={{ marginTop: '0.2rem', color: 'var(--accent-cyan)' }}>
                  {selectedLog.traceID}
                </div>
              </div>
              <div>
                <strong style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>Activity Payload:</strong>
                <pre
                  className="code-font"
                  style={{
                    background: 'rgba(0,0,0,0.5)',
                    padding: '0.75rem',
                    borderRadius: '8px',
                    marginTop: '0.3rem',
                    overflowX: 'auto',
                  }}
                >
                  {JSON.stringify(
                    {
                      event_id: selectedLog.eventID,
                      verb: selectedLog.verb,
                      actor_id: selectedLog.actorID,
                      channel: selectedLog.channel,
                      status: selectedLog.status,
                      latency: selectedLog.latency,
                    },
                    null,
                    2
                  )}
                </pre>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
