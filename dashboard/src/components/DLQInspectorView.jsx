import React, { useState } from 'react'

const initialDLQMessages = [
  {
    id: 'dlq_991023',
    eventID: 'evt_fail_101',
    channel: 'webhook',
    recipientID: 'usr_merchant_44',
    errorMessage: 'webhook returned non-2xx status code: 504 Gateway Timeout',
    retryCount: 3,
    createdAt: '2026-09-06T11:45:10Z',
    replayed: false,
  },
  {
    id: 'dlq_991024',
    eventID: 'evt_fail_102',
    channel: 'push',
    recipientID: 'usr_device_99',
    errorMessage: 'APNs payload expired (410 DeviceTokenNotForTopic)',
    retryCount: 3,
    createdAt: '2026-09-06T11:40:00Z',
    replayed: false,
  },
  {
    id: 'dlq_991025',
    eventID: 'evt_fail_103',
    channel: 'email',
    recipientID: 'usr_invalid_mail',
    errorMessage: 'SendGrid rejected recipient: 550 5.1.1 User unknown',
    retryCount: 3,
    createdAt: '2026-09-06T11:30:12Z',
    replayed: true,
  },
]

export default function DLQInspectorView() {
  const [dlqList, setDlqList] = useState(initialDLQMessages)
  const [replayingAll, setReplayingAll] = useState(false)

  const handleReplaySingle = (id) => {
    setDlqList(
      dlqList.map((m) => {
        if (m.id === id) return { ...m, replayed: true }
        return m
      })
    )
  }

  const handleReplayAll = () => {
    setReplayingAll(true)
    setTimeout(() => {
      setDlqList(dlqList.map((m) => ({ ...m, replayed: true })))
      setReplayingAll(false)
    }, 1000)
  }

  return (
    <div>
      <div className="section-header">
        <div>
          <h1 className="section-title">Dead Letter Queue (DLQ) Inspector</h1>
          <p className="section-desc">
            Inspect failed notification dispatches that exceeded maximum backoff retries and execute 1-click batch replays.
          </p>
        </div>
        <button
          className="btn btn-primary"
          onClick={handleReplayAll}
          disabled={replayingAll}
        >
          {replayingAll ? 'Replaying Batch...' : '🔄 1-Click Batch Replay All'}
        </button>
      </div>

      <div className="glass-card" style={{ padding: 0, overflow: 'hidden' }}>
        <table className="table">
          <thead>
            <tr>
              <th>DLQ ID</th>
              <th>Event ID</th>
              <th>Channel</th>
              <th>Recipient</th>
              <th>Error Details</th>
              <th>Retries</th>
              <th>Timestamp</th>
              <th>Status</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            {dlqList.map((item) => (
              <tr key={item.id}>
                <td className="code-font">{item.id}</td>
                <td className="code-font">{item.eventID}</td>
                <td>
                  <span className="badge badge-cyan">{item.channel}</span>
                </td>
                <td className="code-font">{item.recipientID}</td>
                <td style={{ color: 'var(--accent-rose)', fontSize: '0.8rem' }}>
                  {item.errorMessage}
                </td>
                <td className="code-font">{item.retryCount}</td>
                <td style={{ color: 'var(--text-muted)' }}>{item.createdAt}</td>
                <td>
                  <span className={`badge ${item.replayed ? 'badge-success' : 'badge-error'}`}>
                    {item.replayed ? 'Replayed' : 'Failed'}
                  </span>
                </td>
                <td>
                  {!item.replayed && (
                    <button
                      className="btn btn-secondary"
                      style={{ padding: '0.25rem 0.6rem', fontSize: '0.75rem' }}
                      onClick={() => handleReplaySingle(item.id)}
                    >
                      Replay
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
