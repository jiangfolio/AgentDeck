<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { message } from 'antdv-next'
import { marketCatalogApi, type MarketApp } from '@/api/agent'

const loading = ref(false)
const items = ref<MarketApp[]>([])
const total = ref(0)
const categories = ref<string[]>([])
const query = reactive({
  tab: 'all',
  category: '全部',
  keyword: '',
  page: 1,
  page_size: 20,
  local_only: false,
})

const tagColors = ['blue', 'green', 'orange', 'purple', 'cyan', 'geekblue', 'gold']

function tagColor(name: string) {
  let n = 0
  for (const ch of name) n += ch.charCodeAt(0)
  return tagColors[n % tagColors.length]
}

/**
 * 拉取应用目录。安装按钮保持禁用，这里只展示。
 */
async function load() {
  loading.value = true
  try {
    const data = await marketCatalogApi({
      tab: query.tab === 'settings' ? 'all' : query.tab,
      category: query.category === '全部' ? '' : query.category,
      keyword: query.keyword || undefined,
      page: query.page,
      page_size: query.page_size,
      local_only: query.local_only,
    })
    items.value = data.items || []
    total.value = data.total || 0
    categories.value = data.categories || []
  } catch (e: any) {
    message.error(e?.message || '加载应用目录失败')
  } finally {
    loading.value = false
  }
}

function onTab(key: string) {
  query.tab = key
  query.page = 1
  if (key !== 'settings') void load()
}

function onSearch() {
  query.page = 1
  void load()
}

function onCategory(name: string) {
  query.category = name
  query.page = 1
  void load()
}

onMounted(load)
</script>

<template>
  <div class="ad-page-fill market-page">
    <div class="market-head">
      <a-tabs :active-key="query.tab" @change="(k: string) => onTab(k)">
        <a-tab-pane key="all" tab="全部" />
        <a-tab-pane key="installed" tab="已安装" />
        <a-tab-pane key="upgradable" tab="可升级" />
        <a-tab-pane key="settings" tab="设置" />
      </a-tabs>

      <template v-if="query.tab !== 'settings'">
        <div class="cats">
          <a-button
            size="small"
            :type="query.category === '全部' ? 'primary' : 'default'"
            @click="onCategory('全部')"
          >
            全部
          </a-button>
          <a-button
            v-for="c in categories"
            :key="c"
            size="small"
            :type="query.category === c ? 'primary' : 'default'"
            @click="onCategory(c)"
          >
            {{ c }}
          </a-button>
        </div>

        <div class="ad-page-toolbar">
          <a-button disabled>更新远程应用</a-button>
          <a-button disabled>同步本地应用</a-button>
          <a-input
            v-model:value="query.keyword"
            allow-clear
            placeholder="搜索"
            style="width: 220px; margin-left: auto"
            @press-enter="onSearch"
          />
          <a-button type="primary" @click="onSearch">搜索</a-button>
        </div>
      </template>
    </div>

    <div class="market-body">
      <a-spin v-if="query.tab !== 'settings'" :spinning="loading">
        <div v-if="items.length" class="grid">
          <div v-for="app in items" :key="app.id" class="card">
            <div class="logo">{{ app.name.slice(0, 1) }}</div>
            <div class="body">
              <div class="name">
                <a-tag v-if="app.installed" color="success">已安装</a-tag>
                <span>{{ app.name }}</span>
              </div>
              <div class="summary">{{ app.summary }}</div>
              <div class="foot">
                <a-tag :color="tagColor(app.category)">{{ app.category }}</a-tag>
                <a-button size="small" disabled>安装</a-button>
              </div>
            </div>
          </div>
        </div>
        <a-empty v-else description="没有匹配的应用" />
      </a-spin>
      <div v-else class="settings">
        当前目录是随程序发布的本地静态数据，用来搭好应用市场的页面和筛选。安装、升级和远程同步还没有接上，后续再定怎么装。
      </div>
    </div>

    <div v-if="query.tab !== 'settings'" class="ad-page-pagination">
      <a-pagination
        v-model:current="query.page"
        v-model:page-size="query.page_size"
        :total="total"
        :page-size-options="['20', '40', '60']"
        show-size-changer
        :show-total="(t: number) => `共 ${t} 条`"
        @change="load"
      />
    </div>
  </div>
</template>

<style scoped>
.market-page {
  background: var(--ad-color-bg-container);
}

.market-head {
  flex-shrink: 0;
}

.market-head :deep(.ant-tabs-content-holder) {
  display: none;
}

.market-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

.cats {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}

.card {
  display: flex;
  gap: 12px;
  padding: 16px;
  border: 1px solid var(--ad-color-border-secondary);
  background: var(--ad-color-bg-container);
  transition: border-color 0.2s, box-shadow 0.2s;
}

.body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.card:hover {
  border-color: var(--ad-color-primary);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.logo {
  width: 48px;
  height: 48px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #e6f4ff;
  color: #1677ff;
  font-size: 20px;
  font-weight: 600;
}

.name {
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 8px;
}

.summary {
  margin: 6px 0 10px;
  color: var(--ad-color-text-secondary);
  font-size: 13px;
  min-height: 36px;
}

.foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: auto;
}

.settings {
  padding: 24px 8px;
  color: var(--ad-color-text-secondary);
  line-height: 1.7;
}
</style>
