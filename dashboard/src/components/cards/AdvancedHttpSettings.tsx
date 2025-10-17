import { useState } from 'react'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  Plus,
  Trash,
  ChevronDown,
  Settings,
  Route as RouteIcon,
  Globe,
} from 'lucide-react'
import { Route, ProxyRule, ProxySettings } from '../../lib/api'

type AdvancedHttpSettingsProps = {
  route: Route
  updateRoute: (route: Route) => void
}

export const AdvancedHttpSettings = ({ route, updateRoute }: AdvancedHttpSettingsProps) => {
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [showRules, setShowRules] = useState(false)
  const [showHeaders, setShowHeaders] = useState(false)

  const proxySettings = route.proxy_settings || {}
  const rules = proxySettings.rules || []
  const customHeaders = proxySettings.custom_headers || { add: {}, remove: [], set: {} }

  const updateProxySettings = (updates: Partial<ProxySettings>) => {
    updateRoute({
      ...route,
      proxy_settings: {
        ...proxySettings,
        ...updates,
      },
    })
  }

  const addRule = () => {
    const newRule: ProxyRule = {
      path: '/api/',
      // Leave target_host and target_port undefined to use defaults
      strip_path: false,
      // rewrite is optional and only shown when strip_path is true
    }
    updateProxySettings({
      rules: [...rules, newRule],
    })
  }

  const updateRule = (index: number, updates: Partial<ProxyRule>) => {
    const newRules = [...rules]
    newRules[index] = { ...newRules[index], ...updates }
    updateProxySettings({ rules: newRules })
  }

  const removeRule = (index: number) => {
    updateProxySettings({
      rules: rules.filter((_, i) => i !== index),
    })
  }

  const addHeader = (type: 'add' | 'set') => {
    const newHeaders = { ...customHeaders }
    const key = `header-${Date.now()}`
    newHeaders[type] = { ...newHeaders[type], [key]: '' }
    updateProxySettings({ custom_headers: newHeaders })
  }

  const updateHeaderKey = (oldKey: string, newKey: string, type: 'add' | 'set') => {
    const newHeaders = { ...customHeaders }
    const value = newHeaders[type]?.[oldKey] || ''
    if (newHeaders[type]) {
      delete newHeaders[type][oldKey]
      newHeaders[type][newKey] = value
    }
    updateProxySettings({ custom_headers: newHeaders })
  }

  const updateHeaderValue = (key: string, value: string, type: 'add' | 'set') => {
    const newHeaders = { ...customHeaders }
    if (newHeaders[type]) {
      newHeaders[type][key] = value
    }
    updateProxySettings({ custom_headers: newHeaders })
  }

  const removeHeader = (key: string, type: 'add' | 'set') => {
    const newHeaders = { ...customHeaders }
    if (newHeaders[type]) {
      delete newHeaders[type][key]
    }
    updateProxySettings({ custom_headers: newHeaders })
  }

  const addRemoveHeader = () => {
    const newHeaders = { ...customHeaders }
    newHeaders.remove = [...(newHeaders.remove || []), '']
    updateProxySettings({ custom_headers: newHeaders })
  }

  const updateRemoveHeader = (index: number, value: string) => {
    const newHeaders = { ...customHeaders }
    const newRemove = [...(newHeaders.remove || [])]
    newRemove[index] = value
    newHeaders.remove = newRemove
    updateProxySettings({ custom_headers: newHeaders })
  }

  const removeRemoveHeader = (index: number) => {
    const newHeaders = { ...customHeaders }
    newHeaders.remove = (newHeaders.remove || []).filter((_, i) => i !== index)
    updateProxySettings({ custom_headers: newHeaders })
  }

  if (route.type !== 'http' && route.type !== 'https') {
    return null
  }

  return (
    <Card className="mt-4">
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center gap-2 text-lg">
          <Settings className="h-5 w-5" />
          Advanced HTTP/HTTPS Settings
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Basic Proxy Settings */}
        <Collapsible open={showAdvanced} onOpenChange={setShowAdvanced}>
          <CollapsibleTrigger asChild>
            <Button variant="ghost" className="w-full justify-between h-auto">
              <span className="flex items-center gap-2">
                <Globe className="h-4 w-4" />
                Proxy Configuration
              </span>
              <ChevronDown className={`h-4 w-4 transition-transform ${showAdvanced ? 'rotate-180' : ''}`} />
            </Button>
          </CollapsibleTrigger>
          <CollapsibleContent className="space-y-4 mt-4">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <Label htmlFor="timeout">Timeout (seconds)</Label>
                <Input
                  id="timeout"
                  type="number"
                  value={proxySettings.timeout || 30}
                  onChange={(e) => updateProxySettings({ timeout: parseInt(e.target.value) || 30 })}
                />
              </div>
              <div>
                <Label htmlFor="retry">Retry Attempts</Label>
                <Input
                  id="retry"
                  type="number"
                  value={proxySettings.retry_attempts || 3}
                  onChange={(e) => updateProxySettings({ retry_attempts: parseInt(e.target.value) || 3 })}
                />
              </div>
              <div className="flex flex-col gap-2">
                <Label>Options</Label>
                <div className="space-y-2">
                  <div className="flex items-center space-x-2">
                    <Switch
                      checked={proxySettings.buffer_requests || false}
                      onCheckedChange={(checked) => updateProxySettings({ buffer_requests: checked })}
                    />
                    <Label className="text-sm">Buffer Requests</Label>
                  </div>
                  <div className="flex items-center space-x-2">
                    <Switch
                      checked={proxySettings.preserve_host || false}
                      onCheckedChange={(checked) => updateProxySettings({ preserve_host: checked })}
                    />
                    <Label className="text-sm">Preserve Host</Label>
                  </div>
                  <div className="flex items-center space-x-2">
                    <Switch
                      checked={proxySettings.follow_redirects || false}
                      onCheckedChange={(checked) => updateProxySettings({ follow_redirects: checked })}
                    />
                    <Label className="text-sm">Follow Redirects</Label>
                  </div>
                </div>
              </div>
            </div>
          </CollapsibleContent>
        </Collapsible>

        {/* Path-based Routing Rules */}
        <Collapsible open={showRules} onOpenChange={setShowRules}>
          <CollapsibleTrigger asChild>
            <Button variant="ghost" className="w-full justify-between">
              <span className="flex items-center gap-2">
                <RouteIcon className="h-4 w-4" />
                Path-based Routing Rules ({rules.length})
              </span>
              <ChevronDown className={`h-4 w-4 transition-transform ${showRules ? 'rotate-180' : ''}`} />
            </Button>
          </CollapsibleTrigger>
          <CollapsibleContent className="space-y-4 mt-4">
            <div className="text-sm text-muted-foreground">
              Create nginx-style routing rules to route different paths to different backends.
            </div>
            
            {rules.map((rule, index) => (
              <Card key={`rule-${index}-${rule.path}`} className="p-4">
                <div className="space-y-3">
                  <div className="flex items-center justify-between">
                    <Label className="text-sm font-medium">Rule {index + 1}</Label>
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => removeRule(index)}
                    >
                      <Trash className="h-4 w-4" />
                    </Button>
                  </div>
                  
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                    <div>
                      <Label htmlFor={`path-${index}`}>Path Pattern *</Label>
                      <Input
                        id={`path-${index}`}
                        value={rule.path}
                        onChange={(e) => updateRule(index, { path: e.target.value })}
                        placeholder="/api/"
                        className="font-mono"
                      />
                      <div className="text-xs text-muted-foreground mt-1">
                        e.g., /api/, /static/, /ws/
                      </div>
                    </div>
                    
                    <div>
                      <Label htmlFor={`host-${index}`}>Target Host (optional)</Label>
                      <Input
                        id={`host-${index}`}
                        value={rule.target_host || ''}
                        onChange={(e) => updateRule(index, { target_host: e.target.value })}
                        placeholder="Leave empty for default"
                      />
                      <div className="text-xs text-muted-foreground mt-1">
                        Defaults to main machine
                      </div>
                    </div>
                    
                    <div>
                      <Label htmlFor={`port-${index}`}>Target Port (optional)</Label>
                      <Input
                        id={`port-${index}`}
                        type="number"
                        value={rule.target_port || ''}
                        onChange={(e) => updateRule(index, { target_port: parseInt(e.target.value) || undefined })}
                        placeholder="8080"
                      />
                      <div className="text-xs text-muted-foreground mt-1">
                        Defaults to main port
                      </div>
                    </div>
                  </div>
                  
                  <div className="space-y-3">
                    <div className="flex items-center space-x-2">
                      <Switch
                        checked={rule.strip_path || false}
                        onCheckedChange={(checked) => updateRule(index, { strip_path: checked })}
                      />
                      <Label className="text-sm">Strip matched path from request</Label>
                    </div>
                    
                    {rule.strip_path && (
                      <div>
                        <Label htmlFor={`rewrite-${index}`} className="text-sm">
                          Rewrite Path (optional)
                        </Label>
                        <Input
                          id={`rewrite-${index}`}
                          value={rule.rewrite || ''}
                          onChange={(e) => updateRule(index, { rewrite: e.target.value })}
                          placeholder="/v1/"
                          className="font-mono mt-1"
                        />
                        <div className="text-xs text-muted-foreground mt-1">
                          {rule.path && rule.rewrite ? (
                            <>Example: <code className="bg-muted px-1 rounded">{rule.path}users</code> → <code className="bg-muted px-1 rounded">{rule.rewrite}users</code></>
                          ) : rule.path ? (
                            <>Example: <code className="bg-muted px-1 rounded">{rule.path}users</code> → <code className="bg-muted px-1 rounded">/users</code></>
                          ) : (
                            "Prepend this path after stripping the matched prefix"
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              </Card>
            ))}
            
            <Button onClick={addRule} className="w-full" variant="outline">
              <Plus className="h-4 w-4 mr-2" />
              Add Routing Rule
            </Button>
          </CollapsibleContent>
        </Collapsible>

        {/* Custom Headers */}
        <Collapsible open={showHeaders} onOpenChange={setShowHeaders}>
          <CollapsibleTrigger asChild>
            <Button variant="ghost" className="w-full justify-between">
              <span className="flex items-center gap-2">
                <Settings className="h-4 w-4" />
                Custom Headers
              </span>
              <ChevronDown className={`h-4 w-4 transition-transform ${showHeaders ? 'rotate-180' : ''}`} />
            </Button>
          </CollapsibleTrigger>
          <CollapsibleContent className="space-y-4 mt-4">
            {/* Add Headers */}
            <div>
              <div className="flex items-center justify-between mb-3">
                <Label className="text-sm font-medium">Add Headers</Label>
                <Button onClick={() => addHeader('add')} size="sm" variant="outline">
                  <Plus className="h-4 w-4" />
                </Button>
              </div>
              {Object.entries(customHeaders.add || {}).map(([key, value]) => (
                <div key={key} className="flex gap-2 mb-2">
                  <Input
                    placeholder="Header name"
                    value={key}
                    onChange={(e) => updateHeaderKey(key, e.target.value, 'add')}
                  />
                  <Input
                    placeholder="Header value"
                    value={value}
                    onChange={(e) => updateHeaderValue(key, e.target.value, 'add')}
                  />
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => removeHeader(key, 'add')}
                  >
                    <Trash className="h-4 w-4" />
                  </Button>
                </div>
              ))}
            </div>

            {/* Set Headers */}
            <div>
              <div className="flex items-center justify-between mb-3">
                <Label className="text-sm font-medium">Set/Override Headers</Label>
                <Button onClick={() => addHeader('set')} size="sm" variant="outline">
                  <Plus className="h-4 w-4" />
                </Button>
              </div>
              {Object.entries(customHeaders.set || {}).map(([key, value]) => (
                <div key={key} className="flex gap-2 mb-2">
                  <Input
                    placeholder="Header name"
                    value={key}
                    onChange={(e) => updateHeaderKey(key, e.target.value, 'set')}
                  />
                  <Input
                    placeholder="Header value"
                    value={value}
                    onChange={(e) => updateHeaderValue(key, e.target.value, 'set')}
                  />
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => removeHeader(key, 'set')}
                  >
                    <Trash className="h-4 w-4" />
                  </Button>
                </div>
              ))}
            </div>

            {/* Remove Headers */}
            <div>
              <div className="flex items-center justify-between mb-3">
                <Label className="text-sm font-medium">Remove Headers</Label>
                <Button onClick={addRemoveHeader} size="sm" variant="outline">
                  <Plus className="h-4 w-4" />
                </Button>
              </div>
              {(customHeaders.remove || []).map((header, index) => (
                <div key={`remove-header-${index}-${header}`} className="flex gap-2 mb-2">
                  <Input
                    placeholder="Header name to remove"
                    value={header}
                    onChange={(e) => updateRemoveHeader(index, e.target.value)}
                  />
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => removeRemoveHeader(index)}
                  >
                    <Trash className="h-4 w-4" />
                  </Button>
                </div>
              ))}
            </div>
          </CollapsibleContent>
        </Collapsible>
      </CardContent>
    </Card>
  )
}