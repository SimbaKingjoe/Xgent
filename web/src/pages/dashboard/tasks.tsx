import { useEffect, useState } from 'react'
import Layout from '../../components/Layout'
import { api, Task, Resource } from '../../lib/api'

interface ReasoningEvent {
  type: string
  content: string
  details?: any
  agent_id?: string
}

export default function TasksPage() {
  const [tasks, setTasks] = useState<Task[]>([])
  const [resources, setResources] = useState<Resource[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [expandedTask, setExpandedTask] = useState<number | null>(null)
  const [form, setForm] = useState({ title: '', prompt: '', resource_name: '', resource_type: 'robot' })
  const [formErrors, setFormErrors] = useState<{[key: string]: string}>({})

  useEffect(() => { 
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [ts, rs] = await Promise.all([api.getTasks(), api.getResources()])
      setTasks(ts)
      setResources(rs)
    } catch (err) { console.error(err) }
    finally { setLoading(false) }
  }

  const validateForm = () => {
    const errors: {[key: string]: string} = {}
    
    if (!form.title.trim()) {
      errors.title = '请输入任务标题'
    }
    if (!form.prompt.trim()) {
      errors.prompt = '请输入任务提示词'
    }
    if (!form.resource_name) {
      errors.resource_name = `请选择${form.resource_type === 'robot' ? '机器人' : '团队'}`
    }
    
    setFormErrors(errors)
    return Object.keys(errors).length === 0
  }

  const handleCreate = async () => {
    if (!validateForm()) return
    
    try {
      await api.createTask(form)
      setShowCreate(false)
      setForm({ title: '', prompt: '', resource_name: '', resource_type: 'robot' })
      setFormErrors({})
      loadData()
    } catch (err: any) {
      setFormErrors({ submit: err.message || '创建任务失败，请稍后重试' })
    }
  }

  const robots = resources.filter(r => r.type === 'Robot')
  const teams = resources.filter(r => r.type === 'Team')

  // Parse event logs from task
  const parseEventLogs = (eventLogs: string): ReasoningEvent[] => {
    if (!eventLogs) return []
    try {
      const lines = eventLogs.split('\n').filter(l => l.trim())
      const events: ReasoningEvent[] = []
      for (const line of lines) {
        try {
          const event = JSON.parse(line)
          if (event.type && 
              event.type !== 'debug' && 
              event.type !== 'debug_event' &&
              event.type !== 'content' &&
              event.type !== 'run_content') {
            events.push(event)
          }
        } catch (e) {
          // Skip non-JSON lines
        }
      }
      return events
    } catch (e) {
      return []
    }
  }

  const getEventIcon = (type: string) => {
    if (type === 'error') return '❌'
    if (type === 'warning') return '⚠️'
    if (type === 'started') return '▶️'
    if (type.includes('run_started')) return '🚀'
    if (type.includes('run_completed') || type.includes('completed')) return '✅'
    if (type.includes('tool')) return '🔧'
    if (type.includes('reasoning')) return '💭'
    if (type.includes('thinking')) return '🧠'
    if (type.includes('content')) return '💬'
    if (type.includes('mcp')) return '🔌'
    return '📍'
  }

  if (loading) return <Layout><div className="text-center py-8">Loading...</div></Layout>

  return (
    <Layout>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Tasks</h1>
        <button onClick={() => setShowCreate(true)} className="btn-primary">+ New Task</button>
      </div>

      <div className="space-y-4">
        {tasks.map((task) => {
          const events = parseEventLogs((task as any).event_logs || '')
          const isExpanded = expandedTask === task.id
          
          return (
            <div key={task.id} className="card">
              <div className="flex justify-between items-start">
                <div className="flex-1">
                  <h3 className="font-semibold">{task.title}</h3>
                  <p className="text-gray-500 text-sm mt-1">{task.prompt}</p>
                </div>
                <span className={`px-2 py-1 rounded text-sm ${
                  task.status === 'completed' ? 'bg-green-100 text-green-700' :
                  task.status === 'running' ? 'bg-blue-100 text-blue-700' :
                  task.status === 'failed' ? 'bg-red-100 text-red-700' : 'bg-gray-100 text-gray-700'
                }`}>{task.status}</span>
              </div>
              
              {task.status === 'running' && (
                <div className="mt-3 h-2 bg-gray-200 rounded-full overflow-hidden">
                  <div className="h-full bg-indigo-500 transition-all" style={{ width: `${task.progress}%` }} />
                </div>
              )}
              
              {task.status === 'failed' && (task as any).error && (
                <div className="mt-3 p-3 bg-red-50 border border-red-200 rounded text-sm">
                  <span className="text-red-700 font-semibold">Error: </span>
                  <span className="text-red-600">{(task as any).error}</span>
                </div>
              )}
              
              {task.result && (
                <div className="mt-3 p-3 bg-gray-50 rounded text-sm">
                  <strong>Result:</strong>
                  <pre className="mt-1 whitespace-pre-wrap">{task.result}</pre>
                </div>
              )}
              
              {/* Reasoning Process - click to expand */}
              {events.length > 0 && (
                <div className="mt-3">
                  <button 
                    onClick={() => setExpandedTask(isExpanded ? null : task.id)}
                    className="text-sm text-indigo-600 hover:text-indigo-700 font-medium flex items-center gap-1"
                  >
                    {isExpanded ? '▼' : '▶'} 推理过程 ({events.length} 步骤)
                  </button>
                  
                  {isExpanded && (
                    <div className="mt-3 space-y-2 max-h-96 overflow-y-auto">
                      {events.map((event, idx) => {
                        const isReasoning = event.type === 'reasoning'
                        const details = event.details || {}
                        const agentId = event.agent_id || details.agent_id
                        
                        return (
                          <div key={idx} className={`flex gap-2 text-sm p-3 rounded ${
                            isReasoning ? 'bg-blue-50 border border-blue-200' : 'bg-gray-50'
                          }`}>
                            <span className="text-lg">{getEventIcon(event.type)}</span>
                            <div className="flex-1">
                              <div className="font-medium text-gray-700 flex items-center gap-2">
                                {event.type.replace(/_/g, ' ').toUpperCase()}
                                {agentId && <span className="px-2 py-0.5 bg-purple-100 text-purple-700 text-xs rounded">{agentId}</span>}
                              </div>
                              
                              {event.content && <div className="text-gray-600 mt-1">{event.content}</div>}
                              
                              {isReasoning && details.title && (
                                <div className="mt-2 space-y-1">
                                  <div className="font-semibold text-orange-700">📋 {details.title}</div>
                                  {details.action && <div className="text-gray-700"><strong>Action:</strong> {details.action}</div>}
                                  {details.reasoning && <div className="text-gray-600"><strong>Reasoning:</strong> {details.reasoning}</div>}
                                </div>
                              )}
                              
                              {!isReasoning && details.tool_name && (
                                <div className="text-xs text-gray-500 mt-1">
                                  🔧 Tool: {details.tool_name}
                                </div>
                              )}
                            </div>
                          </div>
                        )
                      })}
                    </div>
                  )}
                </div>
              )}
              
              <div className="mt-3 text-xs text-gray-400">Created: {new Date(task.created_at).toLocaleString()}</div>
            </div>
          )
        })}
      </div>

      {tasks.length === 0 && <div className="text-center py-12 text-gray-500">No tasks yet.</div>}

      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-xl p-6 w-full max-w-lg">
            <h2 className="text-xl font-bold mb-4">New Task</h2>
            <div className="mb-4">
              <label className="block text-sm font-medium mb-1">任务标题 <span className="text-red-500">*</span></label>
              <input 
                type="text" 
                value={form.title} 
                onChange={(e) => { setForm({ ...form, title: e.target.value }); setFormErrors(prev => ({...prev, title: ''})) }}
                className={`input ${formErrors.title ? 'border-red-500' : ''}`}
                placeholder="例如：代码审查任务"
              />
              {formErrors.title && <p className="text-red-500 text-sm mt-1">{formErrors.title}</p>}
            </div>
            <div className="mb-4">
              <label className="block text-sm font-medium mb-1">类型</label>
              <select value={form.resource_type} onChange={(e) => { setForm({ ...form, resource_type: e.target.value, resource_name: '' }); setFormErrors(prev => ({...prev, resource_name: ''})) }} className="input">
                <option value="robot">机器人 (单个Agent)</option>
                <option value="team">团队 (多Agent协作)</option>
              </select>
            </div>
            <div className="mb-4">
              <label className="block text-sm font-medium mb-1">
                {form.resource_type === 'robot' ? '选择机器人' : '选择团队'} <span className="text-red-500">*</span>
              </label>
              <select 
                value={form.resource_name} 
                onChange={(e) => { setForm({ ...form, resource_name: e.target.value }); setFormErrors(prev => ({...prev, resource_name: ''})) }}
                className={`input ${formErrors.resource_name ? 'border-red-500' : ''}`}
              >
                <option value="">-- 请选择 --</option>
                {(form.resource_type === 'robot' ? robots : teams).map((r) => <option key={r.id} value={r.name}>{r.name}</option>)}
              </select>
              {formErrors.resource_name && <p className="text-red-500 text-sm mt-1">{formErrors.resource_name}</p>}
            </div>
            <div className="mb-4">
              <label className="block text-sm font-medium mb-1">任务提示词 <span className="text-red-500">*</span></label>
              <textarea 
                value={form.prompt} 
                onChange={(e) => { setForm({ ...form, prompt: e.target.value }); setFormErrors(prev => ({...prev, prompt: ''})) }}
                className={`input ${formErrors.prompt ? 'border-red-500' : ''}`}
                rows={4}
                placeholder="描述你想让AI完成的任务，例如：请分析这段代码的性能问题并给出优化建议"
              />
              {formErrors.prompt && <p className="text-red-500 text-sm mt-1">{formErrors.prompt}</p>}
            </div>
            {formErrors.submit && (
              <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-sm text-red-600">
                ❌ {formErrors.submit}
              </div>
            )}
            <div className="flex gap-2 justify-end">
              <button onClick={() => { setShowCreate(false); setFormErrors({}) }} className="btn-secondary">取消</button>
              <button onClick={handleCreate} className="btn-primary">创建任务</button>
            </div>
          </div>
        </div>
      )}
    </Layout>
  )
}
