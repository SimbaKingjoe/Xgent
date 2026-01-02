import { useEffect, useState } from 'react'
import Layout from '../../components/Layout'
import { api, Resource, Workspace } from '../../lib/api'

const RESOURCE_TYPES = ['Soul', 'Mind', 'Craft', 'Robot', 'Team']

const TEMPLATES: Record<string, string> = {
  Soul: `apiVersion: xgent.ai/v1
kind: Soul
metadata:
  name: my-assistant          # 唯一名称
spec:
  personality: |              # Agent 人格描述
    You are a helpful AI assistant.
    你是一个乐于助人的AI助手。`,
  Mind: `apiVersion: xgent.ai/v1
kind: Mind
metadata:
  name: gemini-flash          # 唯一名称
spec:
  # provider 可选: ollama, gemini, groq, openai, deepseek, together, openrouter
  provider: gemini            # 模型提供商
  model_id: gemini-2.0-flash-exp  # 模型ID (免费)
  api_key: YOUR_GEMINI_API_KEY    # 从 https://aistudio.google.com/apikey 获取
  # 其他可用模型: gemini-1.5-flash, gemini-1.5-pro
  # temperature: 0.7          # 温度 0-1
  # max_tokens: 4096          # 最大token数`,
  Robot: `apiVersion: xgent.ai/v1
kind: Robot
metadata:
  name: my-robot              # 唯一名称
spec:
  soul: my-assistant          # 引用 Soul 名称
  mind: gemini-flash          # 引用 Mind 名称
  # craft: my-craft           # 可选: 引用 Craft 名称`,
  Team: `apiVersion: xgent.ai/v1
kind: Team
metadata:
  name: my-team               # 唯一名称
spec:
  # mode 可选: coordinate(协调), collaborate(协作), route(路由)
  mode: coordinate            # 协作模式
  leader: my-robot            # 队长 Robot 名称
  members:                    # 成员 Robot 列表
    - member-robot`,
  Craft: `apiVersion: xgent.ai/v1
kind: Craft
metadata:
  name: default-craft         # 唯一名称
spec:
  tools: []                   # MCP工具列表
  # mcp_servers:              # MCP服务器配置
  #   - name: filesystem
  #     command: npx
  #     args: ["-y", "@anthropic/mcp-server-filesystem", "/tmp"]`,
}

export default function ResourcesPage() {
  const [resources, setResources] = useState<Resource[]>([])
  const [workspaces, setWorkspaces] = useState<Workspace[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [editingResource, setEditingResource] = useState<Resource | null>(null)
  const [selectedType, setSelectedType] = useState('Soul')
  const [form, setForm] = useState({ name: '', spec: TEMPLATES.Soul })

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [rs, ws] = await Promise.all([api.getResources(), api.getWorkspaces()])
      setResources(rs)
      setWorkspaces(ws)
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async () => {
    if (!workspaces.length) {
      alert('Please create a workspace first')
      return
    }
    if (!form.name.trim()) {
      alert('请填写资源名称 (Name)')
      return
    }
    if (!selectedType) {
      alert('请选择资源类型 (Type)')
      return
    }
    try {
      const payload = {
        workspace_id: workspaces[0].id,
        name: form.name.trim(),
        type: selectedType,
        spec: form.spec,
      }
      console.log('Creating resource with payload:', payload)
      await api.createResource(payload)
      setShowCreate(false)
      setSelectedType('Soul')
      setForm({ name: '', spec: TEMPLATES.Soul })
      loadData()
    } catch (err: any) {
      console.error('Create resource error:', err)
      alert(err.message)
    }
  }

  const handleEdit = (resource: Resource) => {
    setEditingResource(resource)
    setSelectedType(resource.type)
    setForm({ name: resource.name, spec: resource.spec })
  }

  const handleUpdate = async () => {
    if (!editingResource) return
    if (!form.name.trim()) {
      alert('请填写资源名称 (Name)')
      return
    }
    if (!selectedType) {
      alert('资源类型缺失，请重新打开编辑')
      return
    }
    try {
      const payload = {
        name: form.name.trim(),
        type: selectedType,
        spec: form.spec,
      }
      console.log('Updating resource with payload:', payload)
      await api.updateResource(editingResource.id, payload)
      setEditingResource(null)
      setSelectedType('Soul')
      setForm({ name: '', spec: TEMPLATES.Soul })
      loadData()
    } catch (err: any) {
      console.error('Update resource error:', err)
      alert(err.message)
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this resource?')) return
    try {
      await api.deleteResource(id)
      loadData()
    } catch (err: any) {
      alert(err.message)
    }
  }

  if (loading) return <Layout><div className="text-center py-8">Loading...</div></Layout>

  return (
    <Layout>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Resources</h1>
        <button onClick={() => setShowCreate(true)} className="btn-primary">+ Create</button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {resources.map((resource) => (
          <div key={resource.id} className="card">
            <div className="flex justify-between items-start mb-2">
              <span className="px-2 py-1 bg-indigo-100 text-indigo-700 text-sm rounded">{resource.type}</span>
              <div className="flex gap-2">
                <button onClick={() => handleEdit(resource)} className="text-blue-500 hover:text-blue-700">✏️</button>
                <button onClick={() => handleDelete(resource.id)} className="text-red-500 hover:text-red-700">🗑️</button>
              </div>
            </div>
            <h3 className="font-semibold">{resource.name}</h3>
            <pre className="mt-2 text-xs bg-gray-50 p-2 rounded overflow-auto max-h-32">{resource.spec}</pre>
          </div>
        ))}
      </div>

      {resources.length === 0 && (
        <div className="text-center py-12 text-gray-500">No resources yet. Create your first resource!</div>
      )}

      {(showCreate || editingResource) && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-xl p-6 w-full max-w-2xl max-h-[90vh] overflow-auto">
            <h2 className="text-xl font-bold mb-4">{editingResource ? 'Edit Resource' : 'Create Resource'}</h2>
            
            {!editingResource && (
              <div className="mb-4">
                <label className="block text-sm font-medium mb-1">Type</label>
                <div className="flex gap-2 flex-wrap">
                  {RESOURCE_TYPES.map((type) => (
                    <button
                      key={type}
                      onClick={() => { setSelectedType(type); setForm({ ...form, spec: TEMPLATES[type] }) }}
                      className={`px-3 py-1 rounded ${selectedType === type ? 'bg-indigo-500 text-white' : 'bg-gray-100'}`}
                    >
                      {type}
                    </button>
                  ))}
                </div>
              </div>
            )}
            {editingResource && (
              <div className="mb-4">
                <label className="block text-sm font-medium mb-1">Type</label>
                <div className="px-3 py-1 bg-gray-100 text-gray-600 rounded inline-block">{selectedType}</div>
              </div>
            )}

            <div className="mb-4">
              <label className="block text-sm font-medium mb-1">Name</label>
              <input type="text" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} className="input" placeholder="resource-name" />
            </div>

            <div className="mb-4">
              <label className="block text-sm font-medium mb-1">YAML Spec</label>
              <textarea value={form.spec} onChange={(e) => setForm({ ...form, spec: e.target.value })} className="input font-mono text-sm" rows={12} />
            </div>

            <div className="flex gap-2 justify-end">
              <button onClick={() => { setShowCreate(false); setEditingResource(null); setForm({ name: '', spec: TEMPLATES.Soul }) }} className="btn-secondary">Cancel</button>
              <button onClick={editingResource ? handleUpdate : handleCreate} className="btn-primary">{editingResource ? 'Update' : 'Create'}</button>
            </div>
          </div>
        </div>
      )}
    </Layout>
  )
}
