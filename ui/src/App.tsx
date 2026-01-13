import { useState, useEffect } from 'react'
import { Layout, Input, Button, Table, DatePicker, message, Card, Space, Tag, Statistic, Menu, Breadcrumb, theme } from 'antd'
import { SearchOutlined, ReloadOutlined, CloudDownloadOutlined, DashboardOutlined, HistoryOutlined } from '@ant-design/icons'
import { v4 as uuidv4 } from 'uuid'
import dayjs from 'dayjs'
import './App.css'

const { Header, Content, Footer, Sider } = Layout
const { RangePicker } = DatePicker

interface Snapshot {
  ID: number
  CreatedAt: string
  UpdatedAt: string
  DeletedAt: string | null
  Snapshot: string
  Repository: string
  State: string
  StartTime: string
  Indices: any
}

interface IndexSnapshotMap {
  [key: string]: Snapshot
}

interface RestoreTask {
  ID: number
  CreatedAt: string
  UpdatedAt: string
  DeletedAt: string | null
  TaskID: string
  Index: string
  Repository: string
  Snapshot: string
  RestoreNode: string
  Status: string
  CurrentStage: string | null
  Payload: string | null
  ErrorMessage: string | null
  StartedAt: string | null
  FinishedAt: string | null
}

function App() {
  const [collapsed, setCollapsed] = useState(false)
  const [selectedKey, setSelectedKey] = useState('1')
  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken()

  const [loading, setLoading] = useState(false)
  const [indexPattern, setIndexPattern] = useState('')
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null)
  
  const [previewData, setPreviewData] = useState<IndexSnapshotMap | null>(null)
  const [storeSize, setStoreSize] = useState<string>('')
  
  const [selectedIndices, setSelectedIndices] = useState<string[]>([])
  
  const [tasks, setTasks] = useState<RestoreTask[]>([])
  const [tasksLoading] = useState(false)

  const [messageApi, contextHolder] = message.useMessage()

  // Auto refresh tasks
  useEffect(() => {
    fetchTasks()
    const timer = setInterval(fetchTasks, 5000)
    return () => clearInterval(timer)
  }, [])

  const fetchTasks = async () => {
    // Don't set loading state for background refresh to avoid UI flicker
    try {
      const res = await fetch('/tasks')
      if (res.ok) {
        const json = await res.json()
        setTasks(json)
      }
    } catch (err) {
      console.error(err)
    }
  }

  const handlePreview = async () => {
    if (!indexPattern) {
      messageApi.warning('Please enter index name pattern')
      return
    }

    setLoading(true)
    setPreviewData(null)
    setSelectedIndices([])

    try {
      const payload: any = {
        name: [indexPattern]
      }
      if (dateRange) {
        payload.start_at = dateRange[0].format('YYYY-MM-DD HH:mm:ss')
        payload.end_at = dateRange[1].format('YYYY-MM-DD HH:mm:ss')
      }

      const res = await fetch('/preview', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      })

      if (!res.ok) {
        const err = await res.json()
        throw new Error(err.message || res.statusText)
      }

      const json = await res.json()
      setPreviewData(json.index_snapshot)
      setStoreSize(json.store_size)
      messageApi.success('Preview loaded successfully')
    } catch (err: any) {
      messageApi.error(`Preview failed: ${err.message}`)
    } finally {
      setLoading(false)
    }
  }

  const handleRestore = async () => {
    if (selectedIndices.length === 0 || !previewData) {
      messageApi.warning('Please select indices to restore')
      return
    }

    setLoading(true)
    try {
      const tasksToCreate = selectedIndices.map(idx => ({
        task_id: uuidv4(),
        index: idx,
        repository: previewData[idx].Repository,
        snapshot: previewData[idx].Snapshot,
        store_size: storeSize 
      }))

      const res = await fetch('/restoreTask', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tasks: tasksToCreate })
      })

      if (!res.ok) {
        const err = await res.json()
        throw new Error(err.message || JSON.stringify(err))
      }

      messageApi.success(`Successfully submitted ${selectedIndices.length} restore tasks`)
      setSelectedIndices([])
      setPreviewData(null)
      fetchTasks() // Immediate refresh
      // Optional: switch to tasks view
      // setSelectedKey('2') 
    } catch (err: any) {
      messageApi.error(`Restore failed: ${err.message}`)
    } finally {
      setLoading(false)
    }
  }

  const previewColumns = [
    { title: 'Index Name', dataIndex: 'index', key: 'index' },
    { title: 'Snapshot', dataIndex: 'snapshot', key: 'snapshot' },
    { title: 'Repository', dataIndex: 'repository', key: 'repository' },
    { title: 'Snapshot Time', dataIndex: 'startTime', key: 'startTime' },
  ]

  const getPreviewDataSource = () => {
    if (!previewData) return []
    return Object.keys(previewData).map(key => ({
      key: key,
      index: key,
      snapshot: previewData[key].Snapshot,
      repository: previewData[key].Repository,
      startTime: previewData[key].StartTime,
    }))
  }

  const taskColumns = [
    { title: 'Task ID', dataIndex: 'TaskID', key: 'TaskID', width: 150, ellipsis: true },
    { title: 'Index', dataIndex: 'Index', key: 'Index' },
    { title: 'Snapshot', dataIndex: 'Snapshot', key: 'Snapshot' },
    { title: 'Node', dataIndex: 'RestoreNode', key: 'RestoreNode' },
    { 
      title: 'Status', 
      dataIndex: 'Status', 
      key: 'Status',
      render: (status: string) => {
        let color = 'default'
        if (status === 'SUCCESS') color = 'success'
        if (status === 'FAILED' || status === 'TIMEOUT') color = 'error'
        if (status === 'RUNNING') color = 'processing'
        return <Tag color={color}>{status}</Tag>
      }
    },
    { title: 'Created At', dataIndex: 'CreatedAt', key: 'CreatedAt', render: (t: string) => dayjs(t).format('YYYY-MM-DD HH:mm:ss') },
    { title: 'Message', dataIndex: 'ErrorMessage', key: 'ErrorMessage', ellipsis: true },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      {contextHolder}
      <Sider collapsible collapsed={collapsed} onCollapse={(value) => setCollapsed(value)}>
        <div style={{ height: 32, margin: 16, background: 'rgba(255, 255, 255, 0.2)', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'white', fontWeight: 'bold', overflow: 'hidden', whiteSpace: 'nowrap' }}>
           {collapsed ? "LA" : "LogArk"}
        </div>
        <Menu theme="dark" defaultSelectedKeys={['1']} mode="inline" selectedKeys={[selectedKey]} onClick={(e) => setSelectedKey(e.key)} items={[
            { key: '1', icon: <DashboardOutlined />, label: 'Recovery Panel' },
            { key: '2', icon: <HistoryOutlined />, label: 'Task History' },
        ]} />
      </Sider>
      <Layout>
        <Header style={{ padding: 0, background: colorBgContainer, display: 'flex', alignItems: 'center', justifyContent: 'space-between', paddingRight: 24 }}>
           <h2 style={{ margin: '0 24px' }}>LogArk Recovery Platform</h2>
        </Header>
        <Content style={{ margin: '0 16px' }}>
          <Breadcrumb style={{ margin: '16px 0' }} items={[
              { title: 'Home' },
              { title: selectedKey === '1' ? 'Recovery Panel' : 'Task History' },
          ]} />
          <div style={{ padding: 24, minHeight: 360, background: colorBgContainer, borderRadius: borderRadiusLG }}>
            
            {/* View 1: Restore */}
            <div style={{ display: selectedKey === '1' ? 'block' : 'none' }}>
                <Space direction="vertical" size="large" style={{ width: '100%' }}>
                  <Card title="Search & Preview Snapshots" extra={<Statistic title="Total Size" value={storeSize} valueStyle={{ fontSize: 16 }} />}>
                    <Space direction="vertical" style={{ width: '100%' }}>
                      <Space wrap>
                        <Input 
                          placeholder="Index Name (e.g. log-*)" 
                          value={indexPattern}
                          onChange={e => setIndexPattern(e.target.value)}
                          style={{ width: 200 }}
                          prefix={<SearchOutlined />}
                        />
                        <RangePicker 
                          showTime 
                          onChange={(dates) => setDateRange(dates as any)}
                        />
                        <Button type="primary" onClick={handlePreview} loading={loading}>
                          Preview Snapshots
                        </Button>
                      </Space>

                      {previewData && (
                        <>
                          <Table 
                            dataSource={getPreviewDataSource()} 
                            columns={previewColumns} 
                            rowSelection={{
                              selectedRowKeys: selectedIndices,
                              onChange: (keys) => setSelectedIndices(keys as string[])
                            }}
                            pagination={{ pageSize: 5 }}
                            size="small"
                          />
                          <div style={{ textAlign: 'right', marginTop: 16 }}>
                            <Button 
                              type="primary" 
                              danger 
                              icon={<CloudDownloadOutlined />} 
                              onClick={handleRestore}
                              disabled={selectedIndices.length === 0}
                              loading={loading}
                            >
                              Restore Selected ({selectedIndices.length})
                            </Button>
                          </div>
                        </>
                      )}
                    </Space>
                  </Card>
                </Space>
            </div>

            {/* View 2: Tasks */}
            <div style={{ display: selectedKey === '2' ? 'block' : 'none' }}>
               <Card title="Restore Tasks" extra={<Button icon={<ReloadOutlined />} onClick={fetchTasks}>Refresh</Button>}>
                 <Table 
                   dataSource={tasks} 
                   columns={taskColumns} 
                   rowKey="ID"
                   loading={tasksLoading}
                   pagination={{ pageSize: 10 }}
                 />
               </Card>
            </div>

          </div>
        </Content>
        <Footer style={{ textAlign: 'center' }}>
          LogArk Platform ©{new Date().getFullYear()} Created by 404LifeFound
        </Footer>
      </Layout>
    </Layout>
  )
}

export default App
