import React, { useState } from 'react'

const initialWorkflows = [
  {
    id: 'wf_1',
    triggerEvent: 'post.created',
    batchingEnabled: false,
    batchWindowMinutes: 0,
    channels: { webhook: true, push: true, email: false, sms: false },
  },
  {
    id: 'wf_2',
    triggerEvent: 'comment.liked',
    batchingEnabled: true,
    batchWindowMinutes: 5,
    channels: { webhook: false, push: true, email: false, sms: false },
  },
  {
    id: 'wf_3',
    triggerEvent: 'user.followed',
    batchingEnabled: true,
    batchWindowMinutes: 15,
    channels: { webhook: true, push: true, email: true, sms: false },
  },
]

export default function WorkflowBuilderView() {
  const [workflows, setWorkflows] = useState(initialWorkflows)

  const toggleChannel = (wfId, channel) => {
    setWorkflows(
      workflows.map((wf) => {
        if (wf.id === wfId) {
          return {
            ...wf,
            channels: { ...wf.channels, [channel]: !wf.channels[channel] },
          }
        }
        return wf
      })
    )
  }

  const toggleBatching = (wfId) => {
    setWorkflows(
      workflows.map((wf) => {
        if (wf.id === wfId) {
          return { ...wf, batchingEnabled: !wf.batchingEnabled }
        }
        return wf
      })
    )
  }

  return (
    <div>
      <div className="section-header">
        <div>
          <h1 className="section-title">Visual Workflow Builder</h1>
          <p className="section-desc">
            Map ingested trigger events to active dispatch channels and configure sliding-window batch rules.
          </p>
        </div>
        <button className="btn btn-primary">+ Create New Rule</button>
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: '1.25rem' }}>
        {workflows.map((wf) => (
          <div key={wf.id} className="glass-card">
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
                <div style={{ fontSize: '1.2rem' }}>⚡</div>
                <div>
                  <h3 style={{ fontSize: '1rem' }}>Trigger: <span className="code-font" style={{ color: 'var(--primary)' }}>{wf.triggerEvent}</span></h3>
                  <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>Workflow ID: {wf.id}</span>
                </div>
              </div>

              <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
                <span className="badge badge-purple">
                  {wf.batchingEnabled ? `Batched (${wf.batchWindowMinutes}m window)` : 'Real-time Push'}
                </span>
                <button className="btn btn-secondary" onClick={() => toggleBatching(wf.id)}>
                  Toggle Batching
                </button>
              </div>
            </div>

            <div style={{ borderTop: '1px solid var(--border-color)', paddingTop: '1rem', marginTop: '0.5rem' }}>
              <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginBottom: '0.75rem', fontWeight: 600 }}>
                Target Channel Adapters:
              </div>
              <div style={{ display: 'flex', gap: '1rem' }}>
                {Object.keys(wf.channels).map((channel) => (
                  <button
                    key={channel}
                    className={`btn ${wf.channels[channel] ? 'btn-primary' : 'btn-secondary'}`}
                    style={{ padding: '0.4rem 0.85rem', fontSize: '0.8rem' }}
                    onClick={() => toggleChannel(wf.id, channel)}
                  >
                    {wf.channels[channel] ? '✓' : '+'} {channel.toUpperCase()}
                  </button>
                ))}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
