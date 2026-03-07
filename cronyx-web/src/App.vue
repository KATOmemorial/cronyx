<template>
  <div class="app-container">
    <el-card class="box-card" shadow="hover">
      <template #header>
        <div class="card-header">
          <div class="header-title">
            <h2 style="margin:0; color: #409EFF;">🚀 CronyX 2.0 分布式调度中心</h2>
            <el-tag type="success" effect="dark" style="margin-left: 15px;">集群运行中</el-tag>
          </div>
          <el-button type="primary" size="large" @click="dialogVisible = true">
            + 新建任务
          </el-button>
        </div>
      </template>

      <el-table :data="jobs" style="width: 100%" v-loading="loading" stripe border>
        <el-table-column prop="ID" label="ID" width="60" align="center" />
        <el-table-column prop="name" label="任务名称" width="160">
          <template #default="scope">
            <el-tag effect="light">{{ scope.row.name }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="cron_expr" label="Cron 表达式" width="120" align="center" />
        <el-table-column prop="command" label="Shell 命令">
          <template #default="scope">
            <code class="shell-code">{{ scope.row.command }}</code>
          </template>
        </el-table-column>
        <el-table-column label="下次执行时间" width="180" align="center">
          <template #default="scope">
            <i class="el-icon-time"></i> {{ formatTime(scope.row.next_time) }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="scope">
            <el-tag v-if="scope.row.status === 1" type="success" effect="dark">运行中</el-tag>
            <el-tag v-else type="info" effect="dark">已停止</el-tag>
          </template>
        </el-table-column>
        
        <el-table-column label="集群操作" width="180" align="center">
          <template #default="scope">
            <el-button link type="primary" @click="openLogDrawer(scope.row.ID)">
              查看日志
            </el-button>
            <el-button link type="danger" @click="handleKill(scope.row.ID)">
              💀 强杀
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" title="📦 部署新任务到集群" width="500px" destroy-on-close>
      <el-form :model="form" label-width="100px">
        <el-form-item label="任务名称">
          <el-input v-model="form.name" placeholder="例如：自动化渗透分析脚本" />
        </el-form-item>
        <el-form-item label="Cron表达式">
          <el-input v-model="form.cron_expr" placeholder="*/1 * * * *" />
        </el-form-item>
        <el-form-item label="执行命令">
          <el-input v-model="form.command" type="textarea" :rows="3" placeholder="支持复杂 Shell 指令..." />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitJob">发布</el-button>
        </span>
      </template>
    </el-dialog>

    <el-drawer v-model="logDrawer.visible" :title="`任务 [ID: ${logDrawer.jobId}] 执行追踪`" size="50%">
      <div class="terminal-container" v-loading="logDrawer.loading">
        <div v-if="logDrawer.data.length === 0" class="empty-log">暂无执行记录</div>
        <div v-for="log in logDrawer.data" :key="log.ID" class="log-item">
          <div class="log-header">
            <span class="log-time">[{{ formatTime(log.CreatedAt / 1000) }}]</span>
            <span :class="['log-status', log.status === 1 ? 'status-success' : 'status-error']">
              [{{ log.status === 1 ? 'SUCCESS' : 'FAILED' }}]
            </span>
            <span class="log-cost">耗时: {{ log.end_time - log.start_time }}ms</span>
          </div>
          <pre class="log-output">{{ log.output || log.error || 'No output.' }}</pre>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

// 数据定义
const jobs = ref([])
const loading = ref(false)
const dialogVisible = ref(false)

const form = ref({
  name: '',
  cron_expr: '',
  command: '',
  status: 1
})

// 日志抽屉状态管理
const logDrawer = ref({
  visible: false,
  loading: false,
  jobId: null,
  data: []
})

// 1. 获取任务列表 (对接后端 /api/v1/jobs)
const fetchJobs = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/jobs')
    // 根据后端的返回结构可能需要调整，例如 res.data.data.list
    jobs.value = res.data.data.list || res.data.data
  } catch (err) {
    ElMessage.error('获取集群任务失败')
  } finally {
    loading.value = false
  }
}

// 2. 提交任务 (对接后端 /api/v1/job)
const submitJob = async () => {
  if (!form.value.name || !form.value.command) {
    ElMessage.warning('请填写完整任务信息')
    return
  }
  try {
    await axios.post('/api/v1/job', form.value)
    ElMessage.success('任务下发集群成功')
    dialogVisible.value = false
    // 清空表单
    form.value = { name: '', cron_expr: '', command: '', status: 1 }
    fetchJobs()
  } catch (err) {
    ElMessage.error('发布失败')
  }
}

// 3. 强杀任务 (对接后端 /api/v1/job/kill)
const handleKill = (jobId) => {
  ElMessageBox.prompt('请输入要强杀的具体 Task ID (例如 7-1771725720)', '💀 集群精准强杀', {
    confirmButtonText: '执行强杀',
    cancelButtonText: '取消',
    inputPattern: /.+/,
    inputErrorMessage: 'Task ID 不能为空',
    confirmButtonClass: 'el-button--danger'
  }).then(async ({ value }) => {
    try {
      const res = await axios.post('/api/v1/job/kill', { task_id: value })
      if (res.data.code === 0) {
        ElMessage.success(`指令已广播: ${res.data.data}`)
      } else {
        ElMessage.warning(res.data.msg)
      }
    } catch (err) {
      ElMessage.error('RPC 通讯异常，强杀失败')
    }
  }).catch(() => {})
}

// 4. 查看日志抽屉 (对接后端查看日志接口)
const openLogDrawer = async (id) => {
  logDrawer.value.jobId = id
  logDrawer.value.visible = true
  logDrawer.value.loading = true
  logDrawer.value.data = []
  
  try {
    // 这里假设后端加了这样一个获取日志的接口，如果没有，后端需要补充一个 GET 路由
    const res = await axios.get(`/api/v1/job/${id}/logs`)
    logDrawer.value.data = res.data.data || []
  } catch (err) {
    ElMessage.error('拉取执行追踪失败')
  } finally {
    logDrawer.value.loading = false
  }
}

// 时间格式化工具
const formatTime = (timestamp) => {
  if (!timestamp) return '-'
  return new Date(timestamp * 1000).toLocaleString('zh-CN', { hour12: false })
}

// 页面加载时自动运行
onMounted(() => {
  fetchJobs()
})
</script>

<style scoped>
.app-container {
  padding: 40px;
  max-width: 1400px;
  margin: 0 auto;
  background-color: #f5f7fa;
  min-height: 100vh;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.header-title {
  display: flex;
  align-items: center;
}
.shell-code {
  background-color: #282c34;
  color: #abb2bf;
  padding: 4px 8px;
  border-radius: 4px;
  font-family: 'Fira Code', monospace;
  font-size: 13px;
}

/* 暗黑终端风格日志面板 */
.terminal-container {
  background-color: #1e1e1e;
  height: 100%;
  padding: 20px;
  overflow-y: auto;
  color: #d4d4d4;
  font-family: 'Consolas', 'Courier New', monospace;
}
.empty-log {
  text-align: center;
  color: #666;
  margin-top: 50px;
}
.log-item {
  margin-bottom: 20px;
  border-bottom: 1px dashed #333;
  padding-bottom: 10px;
}
.log-header {
  margin-bottom: 8px;
  font-size: 14px;
}
.log-time { color: #569cd6; margin-right: 10px; }
.log-status { font-weight: bold; margin-right: 10px; }
.status-success { color: #4CAF50; }
.status-error { color: #f44336; }
.log-cost { color: #ce9178; }
.log-output {
  margin: 0;
  padding: 10px;
  background-color: #000;
  border-left: 3px solid #569cd6;
  white-space: pre-wrap;
  word-wrap: break-word;
  font-size: 13px;
}
</style>