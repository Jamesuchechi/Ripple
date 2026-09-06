import React, { useState, useEffect } from 'react'

const mockDLQFallback = [
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
]

export default function DLQInspectorView() {
  const [dlqList, setDlqList] = useState([])
  const [loading, setLoading] = useState(true)
  const [replayingAll, setReplayingAll] = useState(false)

  const fetchDLQ = async () => {
    try {
      setLoading(true)
      const res = await fetch('/v1/admin/projects/00000000-0000-0000-0000-000000000001/dlq', {
        headers: { 'X-Project-ID': '00000000-0000-0000-0000-000000000001' },
      })
      if (res.ok) {
        const data = await res.json()
        if (data.messages && data.messages.length > 0) {
          setDlqList(
            data.messages.map((m) => ({
              id: m.id,
              eventID: m.event_id || 'evt_none',
              channel: m.channel,
              recipientID: m.recipient_id,
              errorMessage: m.error_message,
              retryCount: m.retry_count,
              createdAt: m.created_at ? m.created_at.split('T')[0] : 'Today',
              replayed: !!m.replayed_at,
            }))
          )
        } else {
          setDlqList([])
        }
      } else {
        setDlqList(mockDLQFallback)
      }
    } catch (err) {
      console.error('Failed fetching DLQ:', err)
      setDlqList(mockDLQFallback)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchDLQ()
  }, [])

  const handleReplaySingle = async (id) => {
    try {
      const res = await fetch('/v1/admin/projects/00000000-0000-0000-0000-000000000001/dlq/replay', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Project-ID': '00000000-0000-0000-0000-000000000001',
        },
        body: JSON.stringify({ message_id: id }),
      })
      if (res.ok) {
        fetchDLQ()
      }
    } catch (err) {
      console.error('Failed replaying DLQ message:', err)
    }
  }

  const handleReplayAll = async () => {
    setReplayingAll(true)
    for (const item of dlqList.filter((m) => !m.replayed)) {
      await handleReplaySingle(item.id)
    }
    setReplayingAll(false)
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
