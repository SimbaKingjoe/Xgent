import { useState, useRef, useEffect } from 'react'
import Layout from '../../components/Layout'
import { api, Resource } from '../../lib/api'

interface Message { role: 'user' | 'assistant'; content: string }

export default function ChatPage() {
  const [resources, setResources] = useState<Resource[]>([])
  const [selectedRobot, setSelectedRobot] = useState('')
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    api.getResources().then((rs) => {
      setResources(rs)
      const robots = rs.filter((r) => r.type === 'Robot')
      if (robots.length > 0) setSelectedRobot(robots[0].name)
    })
  }, [])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const handleSend = async () => {
    if (!input.trim() || !selectedRobot || loading) return
    const userMessage: Message = { role: 'user', content: input }
    setMessages((prev) => [...prev, userMessage])
    setInput('')
    setLoading(true)

    try {
      const task = await api.createTask({
        title: `Chat: ${input.slice(0, 30)}...`,
        prompt: input,
        resource_name: selectedRobot,
        resource_type: 'robot',
      })
      let result = ''
      for (let i = 0; i < 60; i++) {
        await new Promise((r) => setTimeout(r, 1000))
        const updated = await api.getTask(task.id)
        if (updated.status === 'completed') { result = updated.result || 'No response'; break }
        if (updated.status === 'failed') { result = 'Task failed'; break }
      }
      setMessages((prev) => [...prev, { role: 'assistant', content: result }])
    } catch (err: any) {
      setMessages((prev) => [...prev, { role: 'assistant', content: `Error: ${err.message}` }])
    } finally { setLoading(false) }
  }

  const robots = resources.filter((r) => r.type === 'Robot')

  return (
    <Layout>
      <div className="h-[calc(100vh-8rem)] flex flex-col">
        <div className="flex justify-between items-center mb-4">
          <h1 className="text-2xl font-bold">Chat</h1>
          <select value={selectedRobot} onChange={(e) => setSelectedRobot(e.target.value)} className="input w-48">
            {robots.map((robot) => <option key={robot.id} value={robot.name}>{robot.name}</option>)}
          </select>
        </div>

        <div className="flex-1 overflow-auto bg-white rounded-xl border border-gray-200 p-4 mb-4">
          {messages.length === 0 ? (
            <div className="h-full flex items-center justify-center text-gray-400">
              Start a conversation with {selectedRobot || 'a robot'}
            </div>
          ) : (
            <div className="space-y-4">
              {messages.map((msg, idx) => (
                <div key={idx} className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
                  <div className={`max-w-[70%] p-3 rounded-lg ${msg.role === 'user' ? 'bg-indigo-500 text-white' : 'bg-gray-100 text-gray-800'}`}>
                    <pre className="whitespace-pre-wrap font-sans">{msg.content}</pre>
                  </div>
                </div>
              ))}
              {loading && (
                <div className="flex justify-start">
                  <div className="bg-gray-100 p-3 rounded-lg"><span className="animate-pulse">Thinking...</span></div>
                </div>
              )}
              <div ref={messagesEndRef} />
            </div>
          )}
        </div>

        <div className="flex gap-2">
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSend()}
            placeholder="Type a message..."
            className="input flex-1"
            disabled={loading || !selectedRobot}
          />
          <button onClick={handleSend} disabled={loading || !selectedRobot || !input.trim()} className="btn-primary disabled:opacity-50">Send</button>
        </div>
      </div>
    </Layout>
  )
}
