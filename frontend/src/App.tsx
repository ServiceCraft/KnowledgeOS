import { useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate, useNavigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Toaster } from '@/components/ui/sonner';
import { TooltipProvider } from '@/components/ui/tooltip';
import { AppLayout } from '@/components/layout/AppLayout';
import { ProtectedRoute } from '@/components/shared/ProtectedRoute';
import { CompanyContextRoute } from '@/components/shared/CompanyContextRoute';
import { setAuthFailureHandler } from '@/api/client';
import { useAuthStore } from '@/stores/authStore';
import { LoginPage } from '@/pages/LoginPage';
import { QAListPage } from '@/pages/qa/QAListPage';
import { QADetailPage } from '@/pages/qa/QADetailPage';
import { ThemesPage } from '@/pages/themes/ThemesPage';
import { PricingPage } from '@/pages/pricing/PricingPage';
import { ArticleListPage } from '@/pages/articles/ArticleListPage';
import { ArticleCreatePage } from '@/pages/articles/ArticleCreatePage';
import { ArticleDetailPage } from '@/pages/articles/ArticleDetailPage';
import { FAQPage } from '@/pages/faq/FAQPage';
import { SearchPage } from '@/pages/search/SearchPage';
import { UsersPage } from '@/pages/settings/UsersPage';
import { SyncPage } from '@/pages/settings/SyncPage';
import { ExportPage } from '@/pages/settings/ExportPage';
import { BotAdminPage } from '@/pages/settings/BotAdminPage';
import { CompaniesPage } from '@/pages/admin/CompaniesPage';
import { CompanyDetailPage } from '@/pages/admin/CompanyDetailPage';
import { SelectCompanyPage } from '@/pages/SelectCompanyPage';
import { BotPlaygroundPage } from '@/pages/bot/BotPlaygroundPage';
import { HandoffPage } from '@/pages/bot/HandoffPage';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

function AuthFailureWatcher() {
  const navigate = useNavigate();
  const logout = useAuthStore((s) => s.logout);
  useEffect(() => {
    setAuthFailureHandler(() => {
      logout();
      navigate('/login', { replace: true });
    });
  }, [navigate, logout]);
  return null;
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <BrowserRouter>
          <AuthFailureWatcher />
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route element={<ProtectedRoute minimumRole="viewer" />}>
              <Route element={<AppLayout />}>
                <Route element={<ProtectedRoute minimumRole="superadmin" />}>
                  <Route path="/admin/companies" element={<CompaniesPage />} />
                  <Route path="/admin/companies/:id" element={<CompanyDetailPage />} />
                </Route>
                <Route path="/select-company" element={<SelectCompanyPage />} />
                <Route element={<CompanyContextRoute />}>
                  <Route path="/kb" element={<Navigate to="/kb/qa" replace />} />
                  <Route path="/kb/qa" element={<QAListPage />} />
                  <Route path="/kb/qa/:id" element={<QADetailPage />} />
                  <Route path="/kb/themes" element={<ThemesPage />} />
                  <Route path="/kb/pricing" element={<PricingPage />} />
                  <Route path="/kb/articles" element={<ArticleListPage />} />
                  <Route element={<ProtectedRoute minimumRole="editor" />}>
                    <Route path="/kb/articles/new" element={<ArticleCreatePage />} />
                  </Route>
                  <Route path="/kb/articles/:id" element={<ArticleDetailPage />} />
                  <Route path="/kb/faq" element={<FAQPage />} />
                  <Route path="/kb/search" element={<SearchPage />} />
                  <Route path="/bot/playground" element={<BotPlaygroundPage />} />
                  <Route path="/bot/handoff" element={<HandoffPage />} />
                  <Route element={<ProtectedRoute minimumRole="admin" />}>
                    <Route path="/settings/users" element={<UsersPage />} />
                    <Route path="/settings/sync" element={<SyncPage />} />
                    <Route path="/settings/bot" element={<BotAdminPage />} />
                  </Route>
                  <Route element={<ProtectedRoute minimumRole="superadmin" />}>
                    <Route path="/settings/export" element={<ExportPage />} />
                  </Route>
                </Route>
              </Route>
            </Route>
            <Route path="*" element={<Navigate to="/kb" replace />} />
          </Routes>
        </BrowserRouter>
        <Toaster />
      </TooltipProvider>
    </QueryClientProvider>
  );
}
