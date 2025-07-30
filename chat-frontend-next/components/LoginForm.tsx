"use client";
import { useState } from 'react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import api from '@/lib/api';
import { useRouter } from 'next/navigation';

export default function LoginForm() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  const handleLogin = async () => {
    setError('');
    setLoading(true);
    
    try {
      const res = await api.post('/auth/login', { email, password });
      localStorage.setItem('token', res.data.token);
      router.push('/');
    } catch (error: any) {
      if (error.code === 'ERR_NETWORK') {
        setError('Cannot connect to server. Make sure the backend is running on http://localhost:8080');
      } else if (error.response) {
        setError(error.response?.data?.error || `Login failed: ${error.response.status}`);
      } else {
        setError(`Login failed: ${error.message}`);
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="p-8 rounded-lg bg-white shadow-md w-96">
      <h2 className="text-xl font-bold mb-4">Sign In</h2>
      
      {error && (
        <div className="mb-4 p-3 rounded-lg bg-red-50 border border-red-200">
          <p className="text-red-600 text-sm">{error}</p>
        </div>
      )}
      
      <Input 
        placeholder="Email"
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        className="mb-4"
        disabled={loading}
      />
      <Input 
        type="password"
        placeholder="Password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        className="mb-4"
        disabled={loading}
      />
      <Button 
        onClick={handleLogin} 
        className="w-full" 
        disabled={loading}
      >
        {loading ? 'Signing in...' : 'Login'}
      </Button>
      
      <div className="mt-4 p-3 bg-blue-50 border border-blue-200 rounded-lg">
        <p className="text-blue-700 text-sm font-medium mb-1">Demo Accounts:</p>
        <p className="text-blue-600 text-xs">admin@windgo.com / admin123</p>
        <p className="text-blue-600 text-xs">demo@windgo.com / admin123</p>
      </div>
    </div>
  );
} 