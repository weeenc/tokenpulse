import { createRouter, createWebHistory } from 'vue-router';
import { useAuthStore } from '../stores/auth.js';
import { authNavigation } from '../utils/navigation.js';

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/dashboard' },
    { path: '/login', component: () => import('../views/LoginView.vue'), meta: { public: true } },
    {
      path: '/register',
      component: () => import('../views/RegisterView.vue'),
      meta: { public: true },
    },
    { path: '/device', component: () => import('../views/DeviceAuthView.vue') },
    { path: '/dashboard', component: () => import('../views/DashboardView.vue') },
    { path: '/devices', component: () => import('../views/DevicesView.vue') },
  ],
});

router.beforeEach(async (to) => {
  const auth = useAuthStore();
  if (!auth.checked) await auth.me();
  return authNavigation(
    {
      path: to.path,
      fullPath: to.fullPath,
      isPublic: Boolean(to.meta.public),
      returnUrl: to.query.returnUrl,
    },
    Boolean(auth.user),
  );
});

export default router;
