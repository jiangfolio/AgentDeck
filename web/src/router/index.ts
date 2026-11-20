import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useTabsStore } from '@/stores/tabs'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/auth/LoginView.vue'),
      meta: { public: true, title: '登录' },
    },
    {
      path: '/',
      name: 'Root',
      component: () => import('@/layout/SideLayout.vue'),
      redirect: '/home',
      children: [
        {
          path: 'home',
          name: 'Home',
          component: () => import('@/views/home/HomeView.vue'),
          meta: { title: '首页', affix: true },
        },
        {
          path: 'agent/market',
          name: 'AgentMarket',
          component: () => import('@/views/agent/MarketView.vue'),
          meta: { title: '应用市场' },
        },
        {
          path: 'agent/monitor',
          name: 'AgentMonitor',
          component: () => import('@/views/agent/MonitorView.vue'),
          meta: { title: '监控' },
        },
        {
          path: 'agent/process',
          name: 'AgentProcess',
          component: () => import('@/views/agent/ProcessView.vue'),
          meta: { title: '进程管理' },
        },
        {
          path: 'agent/firewall',
          name: 'AgentFirewall',
          component: () => import('@/views/agent/FirewallView.vue'),
          meta: { title: '防火墙' },
        },
        {
          path: 'agent/disk',
          name: 'AgentDisk',
          component: () => import('@/views/agent/DiskView.vue'),
          meta: { title: '磁盘管理' },
        },
        {
          path: 'agent/cron',
          name: 'AgentCron',
          component: () => import('@/views/agent/CronView.vue'),
          meta: { title: '定时任务' },
        },
        {
          path: 'agent/startup',
          name: 'AgentStartup',
          component: () => import('@/views/agent/StartupView.vue'),
          meta: { title: '开机自启' },
        },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  const userStore = useUserStore()
  if (!to.meta.public) {
    if (!userStore.user) {
      const me = await userStore.fetchMe()
      if (!me) {
        return { name: 'Login', query: { redirect: to.fullPath } }
      }
    }
  } else if (to.name === 'Login' && userStore.user) {
    return { name: 'Home' }
  }

  if (!to.meta.public && to.name) {
    const tabs = useTabsStore()
    tabs.openTab({
      key: String(to.name),
      title: String(to.meta.title || to.name),
      path: to.fullPath,
      closable: !to.meta.affix,
    })
  }
})

export default router
