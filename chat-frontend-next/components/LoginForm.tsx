"use client";
import { useState } from 'react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Visibility, VisibilityOff } from '@mui/icons-material';
import api from '@/lib/api';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/hooks/useAuth';

export default function LoginForm() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const router = useRouter();
  const { login } = useAuth();

  const handleLogin = async () => {
    setError('');
    setLoading(true);
    
    try {
      const res = await api.post('/auth/login', { email, password });
      const { token, user } = res.data;
      
      // Store token and user data
      login(token, user);
      
      // Role-based redirection
      if (user.role === 'admin') {
        router.push('/'); // Admin dashboard
      } else {
        router.push('/chat'); // Regular chat interface
      }
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

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !loading && email && password) {
      handleLogin();
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!loading && email && password) {
      handleLogin();
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
      
      <form onSubmit={handleSubmit}>
        <Input 
          placeholder="Email"
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          onKeyPress={handleKeyPress}
          className="mb-4"
          disabled={loading}
        />
        <div className="relative mb-4">
          <Input 
            type={showPassword ? "text" : "password"}
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            onKeyPress={handleKeyPress}
            className="pr-12"
            disabled={loading}
          />
          <button
            type="button"
            onClick={() => setShowPassword(!showPassword)}
            className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-500 hover:text-gray-700 focus:outline-none"
          >
            {showPassword ? <VisibilityOff fontSize="small" /> : <Visibility fontSize="small" />}
          </button>
        </div>
        <Button 
          type="submit"
          className="w-full" 
          disabled={loading || !email || !password}
        >
          {loading ? 'Signing in...' : 'Login'}
        </Button>
      </form>
      
      <div className="mt-4 p-3 bg-blue-50 border border-blue-200 rounded-lg">
        <p className="text-blue-700 text-sm font-medium mb-1">Demo Accounts:</p>
        <p className="text-blue-600 text-xs"><strong>Admin:</strong> admin@windgo.com / admin123</p>
        <p className="text-blue-600 text-xs"><strong>User:</strong> demo@windgo.com / admin123</p>
        <p className="text-blue-600 text-xs"><strong>User:</strong> test@windgo.com / test123</p>
      </div>
    </div>
  );
} 