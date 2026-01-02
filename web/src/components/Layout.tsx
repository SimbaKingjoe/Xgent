import { useEffect, useState } from 'react'
import { useRouter } from 'next/router'
import Link from 'next/link'
import { api } from '../lib/api'

const navItems = [
  { name: 'Dashboard', href: '/dashboard', icon: '🏠' },
  { name: 'Resources', href: '/dashboard/resources', icon: '📦' },
  { name: 'Tasks', href: '/dashboard/tasks', icon: '📋' },
  { name: 'Chat', href: '/dashboard/chat', icon: '💬' },
]

export default function Layout({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true)
    const token = api.getToken()
    if (!token) {
      router.push('/login')
    }
  }, [router])

  const handleLogout = () => {
    api.clearToken()
    router.push('/login')
  }

  if (!mounted) return null

  return (
    <div className="min-h-screen flex">
      <aside className="w-64 bg-white border-r border-gray-200 p-4 fixed h-full">
        <div className="mb-8">
          <h1 className="text-2xl font-bold text-indigo-500">Xgent</h1>
        </div>
        
        <nav className="space-y-1">
          {navItems.map((item) => (
            <Link key={item.href} href={item.href}>
              <a className={`flex items-center gap-3 px-3 py-2 rounded-lg transition-colors ${
                router.pathname === item.href
                  ? 'bg-indigo-500 text-white'
                  : 'text-gray-600 hover:bg-gray-100'
              }`}>
                <span>{item.icon}</span>
                <span>{item.name}</span>
              </a>
            </Link>
          ))}
        </nav>

        <div className="absolute bottom-4 left-4 right-4">
          <button onClick={handleLogout} className="w-full btn-secondary text-sm">
            Logout
          </button>
        </div>
      </aside>

      <main className="flex-1 p-8 bg-gray-50 ml-64">
        {children}
      </main>
    </div>
  )
}
