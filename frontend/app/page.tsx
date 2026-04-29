'use client'

import { useState, useEffect } from 'react'
import { Loader2, Copy, Check, Zap } from 'lucide-react'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081'

export default function Home() {
  const [url, setUrl] = useState('')
  const [shortUrl, setShortUrl] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [copied, setCopied] = useState(false)
  const [theme, setTheme] = useState<'light' | 'dark'>('light')
  const [urlCount, setUrlCount] = useState(0)

  // Initialize theme from localStorage
  useEffect(() => {
    const savedTheme = (localStorage.getItem('theme') || 'light') as 'light' | 'dark'
    setTheme(savedTheme)
    if (savedTheme === 'dark') {
      document.documentElement.classList.add('dark')
    }
  }, [])

  const toggleTheme = () => {
    const newTheme = theme === 'light' ? 'dark' : 'light'
    setTheme(newTheme)
    localStorage.setItem('theme', newTheme)
    if (newTheme === 'dark') {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
  }

  const handleShorten = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!url.trim()) {
      setError('Please enter a URL')
      return
    }

    try {
      new URL(url)
    } catch {
      setError('Please enter a valid URL')
      return
    }

    setLoading(true)
    setError('')
    setShortUrl('')

    try {
      const response = await fetch(`${API_URL}/`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url }),
      })

      if (!response.ok) {
        throw new Error('Failed to shorten URL')
      }

      const data = await response.json()
      if (data.short) {
        setShortUrl(data.short)
        setUrlCount(prev => prev + 1)
        setUrl('')
      }
    } catch (err) {
      setError('Cannot connect to server. Make sure the API is running on http://localhost:8081')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(shortUrl)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      setError('Failed to copy')
      console.error(err)
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-blue-50 to-slate-100 dark:from-slate-950 dark:via-slate-900 dark:to-slate-950 flex items-center justify-center p-4 transition-colors duration-300">
      <div className="w-full max-w-2xl">
        {/* Header with Theme Toggle */}
        <div className="flex justify-between items-start mb-12">
          <div>
            <h1 className="text-4xl md:text-5xl font-bold text-slate-900 dark:text-white mb-2">
              Link Shortener
            </h1>
            <p className="text-slate-600 dark:text-slate-400 text-sm md:text-base">
              Transform long URLs into short, shareable links instantly
            </p>
          </div>
          <button 
            onClick={toggleTheme}
            className="p-3 rounded-full bg-slate-200 dark:bg-slate-800 hover:bg-slate-300 dark:hover:bg-slate-700 transition-colors duration-200"
          >
            <span className="text-xl">{theme === 'light' ? '🌙' : '☀️'}</span>
          </button>
        </div>

        {/* Main Card */}
        <div className="bg-white dark:bg-slate-900 rounded-2xl shadow-xl dark:shadow-2xl p-8 md:p-10 animate-in fade-in slide-in-from-bottom-3 duration-600 border border-slate-100 dark:border-slate-800 transition-all">
          
          {/* Form */}
          <form onSubmit={handleShorten} className="space-y-6">
            {/* URL Input */}
            <div>
              <label htmlFor="url" className="block text-sm font-semibold text-slate-700 dark:text-slate-200 mb-3 uppercase tracking-wider">
                Paste Your URL
              </label>
              <input 
                type="url" 
                id="url"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder="https://example.com/very/long/url/path" 
                className="w-full px-4 py-3 md:py-4 border-2 border-slate-200 dark:border-slate-700 dark:bg-slate-800 dark:text-white rounded-xl focus:border-blue-500 dark:focus:border-blue-400 focus:outline-none focus:ring-4 focus:ring-blue-100 dark:focus:ring-blue-900 transition-all duration-200 text-base placeholder-slate-400 dark:placeholder-slate-500"
                autoFocus
              />
            </div>

            {/* Action Buttons */}
            <div className="flex flex-col sm:flex-row gap-3">
              <button 
                type="submit"
                disabled={loading}
                className="flex-1 bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 disabled:bg-slate-300 disabled:cursor-not-allowed text-white font-semibold py-3 md:py-4 px-6 rounded-xl transition-all duration-200 transform hover:scale-105 active:scale-100 disabled:scale-100 shadow-md hover:shadow-lg flex items-center justify-center gap-2 uppercase tracking-wide"
              >
                {loading ? (
                  <>
                    <Loader2 className="w-5 h-5 animate-spin" />
                    Generating...
                  </>
                ) : (
                  <>
                    <Zap className="w-5 h-5" />
                    Shorten
                  </>
                )}
              </button>
              
              {shortUrl && (
                <button 
                  type="button"
                  onClick={handleCopy}
                  className={`flex-1 sm:flex-0 sm:min-w-fit px-6 font-semibold py-3 md:py-4 rounded-xl transition-all duration-200 flex items-center justify-center gap-2 uppercase tracking-wide ${
                    copied 
                      ? 'bg-green-500 text-white' 
                      : 'bg-slate-200 dark:bg-slate-700 hover:bg-slate-300 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-200'
                  }`}
                >
                  {copied ? (
                    <>
                      <Check className="w-5 h-5" />
                      Copied!
                    </>
                  ) : (
                    <>
                      <Copy className="w-5 h-5" />
                      Copy
                    </>
                  )}
                </button>
              )}
            </div>
          </form>

          {/* Result Box */}
          {shortUrl && (
            <div className="mt-6 bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-slate-800 dark:to-slate-700 rounded-xl p-6 border-2 border-blue-200 dark:border-blue-900 animate-in fade-in slide-in-from-bottom-2 duration-300">
              <p className="text-sm font-semibold text-slate-600 dark:text-slate-400 uppercase tracking-wider mb-3">
                ✨ Your Shortened Link
              </p>
              <div className="bg-white dark:bg-slate-900 rounded-lg p-4 border border-slate-200 dark:border-slate-700">
                <p className="text-lg md:text-xl font-mono font-semibold text-blue-600 dark:text-blue-400 break-all select-all cursor-pointer hover:text-blue-700 dark:hover:text-blue-300 transition-colors">
                  {shortUrl}
                </p>
              </div>
            </div>
          )}

          {/* Error Message */}
          {error && (
            <div className="mt-6 bg-red-50 dark:bg-red-950 border-2 border-red-200 dark:border-red-800 rounded-xl p-4 text-red-700 dark:text-red-300 font-medium animate-in fade-in slide-in-from-bottom-2 duration-300">
              {error}
            </div>
          )}
        </div>

        {/* Footer Stats */}
        <div className="mt-12 grid grid-cols-3 gap-4 text-center">
          <div className="bg-white dark:bg-slate-900 rounded-lg p-4 shadow-sm border border-slate-100 dark:border-slate-800">
            <p className="text-2xl md:text-3xl font-bold text-blue-600 dark:text-blue-400">{urlCount}</p>
            <p className="text-sm text-slate-600 dark:text-slate-400">URLs Created</p>
          </div>
          <div className="bg-white dark:bg-slate-900 rounded-lg p-4 shadow-sm border border-slate-100 dark:border-slate-800">
            <p className="text-2xl md:text-3xl font-bold text-green-600 dark:text-green-400">∞</p>
            <p className="text-sm text-slate-600 dark:text-slate-400">No Limit</p>
          </div>
          <div className="bg-white dark:bg-slate-900 rounded-lg p-4 shadow-sm border border-slate-100 dark:border-slate-800">
            <p className="text-2xl md:text-3xl font-bold text-purple-600 dark:text-purple-400">⚡</p>
            <p className="text-sm text-slate-600 dark:text-slate-400">Instant</p>
          </div>
        </div>
      </div>
    </div>
  )
}
