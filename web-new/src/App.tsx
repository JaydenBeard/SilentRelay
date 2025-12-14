import { useEffect } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuthStore } from '@/core/store/authStore';
import { useSettingsStore } from '@/core/store/settingsStore';
import { Toaster } from '@/components/ui/toaster';
import LandingPage from '@/pages/Landing';
import AuthPage from '@/pages/Auth';
import ChatPage from '@/pages/Chat';
import PrivacyPolicy from '@/pages/legal/PrivacyPolicy';
import TermsOfService from '@/pages/legal/TermsOfService';
import SecurityPolicy from '@/pages/legal/SecurityPolicy';
import DevShowcase from '@/pages/DevShowcase';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuthStore();

  if (!isAuthenticated) {
    return <Navigate to="/auth" replace />;
  }

  return <>{children}</>;
}

function App() {
  const { isAuthenticated } = useAuthStore();
  const { app, loadPrivacyFromServer } = useSettingsStore();

  // Load privacy settings from server when authenticated
  useEffect(() => {
    if (isAuthenticated) {
      loadPrivacyFromServer();
    }
  }, [isAuthenticated, loadPrivacyFromServer]);

  // Apply theme setting
  useEffect(() => {
    const root = document.documentElement;

    if (app.theme === 'system') {
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      root.classList.toggle('dark', prefersDark);
      root.classList.toggle('light', !prefersDark);
    } else {
      root.classList.toggle('dark', app.theme === 'dark');
      root.classList.toggle('light', app.theme === 'light');
    }
  }, [app.theme]);

  // Apply font size setting
  useEffect(() => {
    const root = document.documentElement;
    root.classList.remove('font-small', 'font-medium', 'font-large');
    root.classList.add(`font-${app.fontSize}`);
  }, [app.fontSize]);

  return (
    <div className="min-h-screen bg-background">
      <Routes>
        <Route
          path="/"
          element={isAuthenticated ? <Navigate to="/chat" replace /> : <LandingPage />}
        />
        <Route
          path="/auth"
          element={isAuthenticated ? <Navigate to="/chat" replace /> : <AuthPage />}
        />
        <Route
          path="/chat"
          element={
            <ProtectedRoute>
              <ChatPage />
            </ProtectedRoute>
          }
        />
        <Route path="/privacy" element={<PrivacyPolicy />} />
        <Route path="/terms" element={<TermsOfService />} />
        <Route path="/security" element={<SecurityPolicy />} />
        <Route path="/dev" element={<DevShowcase />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
      <Toaster />
    </div>
  );
}

export default App;
