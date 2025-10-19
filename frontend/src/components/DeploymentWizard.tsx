import { useState, useEffect } from 'react'
import { useDevices, useValidateDeployment, useRecommendDevice, useCreateDeployment, useDeployment } from '../api/hooks'
import { apiClient } from '../api/client'
import { formatTime, formatRAM, getDependencyIcon, getDependencyName } from '../lib/dependency-utils'
import type { Recipe, DeviceScore, Deployment, DependencyCheckResult } from '../api/types'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from './ui/dialog'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Label } from './ui/label'
import { Checkbox } from './ui/checkbox'
import { CheckCircle, XCircle, AlertCircle, Loader2, Check, Clock, HardDrive, Cpu } from 'lucide-react'
import { toast } from 'sonner'
import { LogViewer } from './LogViewer'

interface DeploymentWizardProps {
  recipe: Recipe
  open: boolean
  onOpenChange: (open: boolean) => void
}

type Step = 'select-device' | 'dependencies' | 'configure' | 'validate' | 'deploy'

export function DeploymentWizard({ recipe, open, onOpenChange }: DeploymentWizardProps) {
  const [currentStep, setCurrentStep] = useState<Step>('select-device')
  const [selectedDeviceId, setSelectedDeviceId] = useState<string>('')
  const [config, setConfig] = useState<Record<string, any>>({})
  const [deploymentId, setDeploymentId] = useState<string>('')
  const [dependencies, setDependencies] = useState<DependencyCheckResult | null>(null)
  const [dependenciesLoading, setDependenciesLoading] = useState(false)
  const [dependenciesError, setDependenciesError] = useState<string | null>(null)

  const { data: devices } = useDevices()
  const { data: deviceScores, isLoading: scoresLoading } = useRecommendDevice(recipe.slug)
  const validateDeployment = useValidateDeployment()
  const createDeployment = useCreateDeployment()
  const { data: deployment } = useDeployment(deploymentId)

  const selectedDevice = devices?.find((d) => d.id === selectedDeviceId)

  // Auto-select best device when scores load
  useEffect(() => {
    if (deviceScores && deviceScores.length > 0 && !selectedDeviceId) {
      // Find the first available device with best score
      const bestDevice = deviceScores.find((s) => s.available)
      if (bestDevice) {
        setSelectedDeviceId(bestDevice.device_id)
        // Initialize config with default values
        const defaultConfig: Record<string, any> = {}
        recipe.config_options?.forEach((option) => {
          defaultConfig[option.name] = option.default
        })
        setConfig(defaultConfig)
      }
    }
  }, [deviceScores, selectedDeviceId, recipe.config_options])

  // Initialize config with default values
  const initializeConfig = () => {
    const defaultConfig: Record<string, any> = {}
    recipe.config_options?.forEach((option) => {
      defaultConfig[option.name] = option.default
    })
    setConfig(defaultConfig)
  }

  const handleDeviceSelect = (deviceId: string) => {
    setSelectedDeviceId(deviceId)
    initializeConfig()
  }

  const handleConfigChange = (name: string, value: any) => {
    setConfig((prev) => ({
      ...prev,
      [name]: value,
    }))
  }

  const handleNext = async () => {
    if (currentStep === 'select-device') {
      if (!selectedDeviceId) {
        toast.error('Please select a device')
        return
      }
      // Check dependencies first, then advance step only if successful
      const success = await checkDependencies()
      if (success) {
        setCurrentStep('dependencies')
      }
    } else if (currentStep === 'dependencies') {
      setCurrentStep('configure')
    } else if (currentStep === 'configure') {
      // Validate required fields
      const missingFields = recipe.config_options
        ?.filter((opt) => opt.required && !config[opt.name])
        .map((opt) => opt.label)

      if (missingFields && missingFields.length > 0) {
        toast.error('Missing required fields', {
          description: missingFields.join(', '),
        })
        return
      }
      setCurrentStep('validate')
      // Trigger validation
      handleValidate()
    } else if (currentStep === 'validate') {
      setCurrentStep('deploy')
    }
  }

  const handleBack = () => {
    if (currentStep === 'dependencies') {
      setCurrentStep('select-device')
    } else if (currentStep === 'configure') {
      setCurrentStep('dependencies')
    } else if (currentStep === 'validate') {
      setCurrentStep('configure')
    } else if (currentStep === 'deploy') {
      setCurrentStep('validate')
    }
  }

  const checkDependencies = async (): Promise<boolean> => {
    if (!selectedDeviceId) return false

    setDependenciesLoading(true)
    setDependenciesError(null)
    try {
      const result = await apiClient.checkRecipeDependencies(recipe.slug, selectedDeviceId)
      setDependencies(result)
      return true
    } catch (err) {
      setDependenciesError(err instanceof Error ? err.message : 'Failed to check dependencies')
      toast.error('Failed to check dependencies', {
        description: err instanceof Error ? err.message : 'Please try again'
      })
      return false
    } finally {
      setDependenciesLoading(false)
    }
  }

  const handleValidate = async () => {
    if (!selectedDeviceId) return

    try {
      await validateDeployment.mutateAsync({
        slug: recipe.slug,
        data: {
          device_id: selectedDeviceId,
          config,
        },
      })
    } catch (error) {
      // Error is handled by the validation result display
    }
  }

  const handleDeploy = async () => {
    if (!selectedDeviceId) return

    try {
      const newDeployment = await createDeployment.mutateAsync({
        recipe_slug: recipe.slug,
        device_id: selectedDeviceId,
        config,
      })

      setDeploymentId(newDeployment.id)
      toast.success('Deployment started successfully')
    } catch (error) {
      toast.error('Failed to start deployment', {
        description: error instanceof Error ? error.message : 'Unknown error',
      })
    }
  }

  const handleClose = () => {
    setCurrentStep('select-device')
    setSelectedDeviceId('')
    setConfig({})
    setDeploymentId('')
    setDependencies(null)
    setDependenciesError(null)
    validateDeployment.reset()
    createDeployment.reset()
    onOpenChange(false)
  }

  const renderStepIndicator = () => {
    const steps = [
      { id: 'select-device', label: 'Device' },
      { id: 'dependencies', label: 'Dependencies' },
      { id: 'configure', label: 'Configure' },
      { id: 'validate', label: 'Validate' },
      { id: 'deploy', label: 'Deploy' },
    ]

    const currentIndex = steps.findIndex((s) => s.id === currentStep)

    return (
      <div className="flex items-center justify-center gap-2 mb-6">
        {steps.map((step, index) => (
          <div key={step.id} className="flex items-center">
            <div
              className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${
                index <= currentIndex
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-muted text-muted-foreground'
              }`}
            >
              {index < currentIndex ? '✓' : index + 1}
            </div>
            {index < steps.length - 1 && (
              <div
                className={`w-12 h-0.5 mx-1 ${
                  index < currentIndex ? 'bg-primary' : 'bg-muted'
                }`}
              />
            )}
          </div>
        ))}
      </div>
    )
  }

  const getRecommendationBadge = (recommendation: DeviceScore['recommendation']) => {
    switch (recommendation) {
      case 'best':
        return <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">🏆 Best Choice</span>
      case 'good':
        return <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800">✅ Good Choice</span>
      case 'acceptable':
        return <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-yellow-100 text-yellow-800">⚠️ Acceptable</span>
      case 'not-recommended':
        return <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-red-100 text-red-800">❌ Not Recommended</span>
    }
  }

  const renderSelectDevice = () => (
    <div className="space-y-4">
      {scoresLoading && (
        <div className="flex items-center justify-center py-4">
          <Loader2 className="w-6 h-6 animate-spin text-primary" />
          <span className="ml-2 text-sm text-muted-foreground">Analyzing devices...</span>
        </div>
      )}

      {!scoresLoading && deviceScores && deviceScores.length > 0 && (
        <div className="space-y-3">
          <Label>Select Target Device</Label>
          <p className="text-sm text-muted-foreground">
            Devices are ranked by suitability for {recipe.name}
          </p>

          {deviceScores.map((score) => {
            const isSelected = score.device_id === selectedDeviceId
            return (
              <div
                key={score.device_id}
                onClick={() => score.available && handleDeviceSelect(score.device_id)}
                className={`border rounded-lg p-4 cursor-pointer transition-all ${
                  isSelected
                    ? 'border-primary bg-primary/5 ring-2 ring-primary/20'
                    : score.available
                    ? 'border-border hover:border-primary/50 hover:bg-muted/50'
                    : 'border-border bg-muted/30 opacity-60 cursor-not-allowed'
                }`}
              >
                <div className="flex items-start justify-between mb-2">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      <h4 className="font-medium">{score.device_name}</h4>
                      {getRecommendationBadge(score.recommendation)}
                    </div>
                    <p className="text-sm text-muted-foreground">{score.device_ip}</p>
                  </div>
                  <div className="text-right">
                    <div className="text-2xl font-bold text-primary">{score.score}</div>
                    <div className="text-xs text-muted-foreground">score</div>
                  </div>
                </div>

                <div className="space-y-1 mt-3">
                  {score.reasons.map((reason, idx) => (
                    <div key={idx} className="text-sm text-muted-foreground flex items-start gap-2">
                      <span className="mt-0.5">•</span>
                      <span>{reason}</span>
                    </div>
                  ))}
                </div>
              </div>
            )
          })}
        </div>
      )}

      {!scoresLoading && (!deviceScores || deviceScores.length === 0) && (
        <div className="text-center py-8 text-muted-foreground">
          <p>No devices available</p>
          <p className="text-sm mt-2">Add devices to your homelab first</p>
        </div>
      )}
    </div>
  )

  const renderConfigure = () => (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        Configure the deployment options for {recipe.name}
      </p>

      {recipe.config_options?.map((option) => (
        <div key={option.name}>
          <Label htmlFor={option.name}>
            {option.label}
            {option.required && <span className="text-red-500 ml-1">*</span>}
          </Label>

          {option.type === 'string' && (
            <Input
              id={option.name}
              type="text"
              value={config[option.name] || ''}
              onChange={(e) => handleConfigChange(option.name, e.target.value)}
              placeholder={option.description}
            />
          )}

          {option.type === 'number' && (
            <Input
              id={option.name}
              type="number"
              value={config[option.name] || ''}
              onChange={(e) => handleConfigChange(option.name, Number(e.target.value))}
              placeholder={option.description}
            />
          )}

          {(option.type === 'password' || option.type === 'secret') && (
            <Input
              id={option.name}
              type="password"
              value={config[option.name] || ''}
              onChange={(e) => handleConfigChange(option.name, e.target.value)}
              placeholder={option.description}
              autoComplete="new-password"
            />
          )}

          {option.type === 'boolean' && (
            <div className="flex items-center space-x-2 mt-2">
              <Checkbox
                id={option.name}
                checked={config[option.name] || false}
                onCheckedChange={(checked: boolean) => handleConfigChange(option.name, checked)}
              />
              <label
                htmlFor={option.name}
                className="text-sm text-muted-foreground cursor-pointer"
              >
                {option.description}
              </label>
            </div>
          )}

          {option.description && option.type !== 'boolean' && (
            <p className="text-xs text-muted-foreground mt-1">
              {option.description}
            </p>
          )}
        </div>
      ))}
    </div>
  )

  const renderValidate = () => {
    const validationResult = validateDeployment.data

    return (
      <div className="space-y-4">
        <p className="text-sm text-muted-foreground">
          Validating deployment to {selectedDevice?.name}...
        </p>

        {validateDeployment.isPending && (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="w-8 h-8 animate-spin text-primary" />
          </div>
        )}

        {validateDeployment.isError && (
          <div className="bg-red-50 border border-red-200 rounded-lg p-4">
            <div className="flex items-start gap-3">
              <XCircle className="w-5 h-5 text-red-600 mt-0.5" />
              <div>
                <div className="font-medium text-red-900">Validation Failed</div>
                <div className="text-sm text-red-700 mt-1">
                  {(validateDeployment.error as Error)?.message || 'Unknown error'}
                </div>
              </div>
            </div>
          </div>
        )}

        {validationResult && (
          <div className="space-y-3">
            {/* Overall Status */}
            <div
              className={`rounded-lg p-4 ${
                validationResult.valid
                  ? 'bg-green-50 border border-green-200'
                  : 'bg-red-50 border border-red-200'
              }`}
            >
              <div className="flex items-start gap-3">
                {validationResult.valid ? (
                  <CheckCircle className="w-5 h-5 text-green-600 mt-0.5" />
                ) : (
                  <XCircle className="w-5 h-5 text-red-600 mt-0.5" />
                )}
                <div>
                  <div
                    className={`font-medium ${
                      validationResult.valid ? 'text-green-900' : 'text-red-900'
                    }`}
                  >
                    {validationResult.valid
                      ? 'Ready to Deploy'
                      : 'Cannot Deploy'}
                  </div>
                </div>
              </div>
            </div>

            {/* Resource Checks */}
            {validationResult.resource_check && (
              <div className="space-y-2">
                <div className="text-sm font-medium">Resource Availability</div>
                <div className="space-y-1">
                  <div className="flex items-center gap-2 text-sm">
                    {validationResult.resource_check.ram_sufficient ? (
                      <CheckCircle className="w-4 h-4 text-green-600" />
                    ) : (
                      <XCircle className="w-4 h-4 text-red-600" />
                    )}
                    <span>
                      RAM: {(validationResult.resource_check.available_ram_mb / 1024).toFixed(1)} GB
                      available (need{' '}
                      {(validationResult.resource_check.required_ram_mb / 1024).toFixed(1)} GB)
                    </span>
                  </div>
                  <div className="flex items-center gap-2 text-sm">
                    {validationResult.resource_check.storage_sufficient ? (
                      <CheckCircle className="w-4 h-4 text-green-600" />
                    ) : (
                      <XCircle className="w-4 h-4 text-red-600" />
                    )}
                    <span>
                      Storage: {validationResult.resource_check.available_storage_gb} GB available
                      (need {validationResult.resource_check.required_storage_gb} GB)
                    </span>
                  </div>
                  <div className="flex items-center gap-2 text-sm">
                    {validationResult.resource_check.docker_installed ? (
                      <CheckCircle className="w-4 h-4 text-green-600" />
                    ) : (
                      <XCircle className="w-4 h-4 text-red-600" />
                    )}
                    <span>Docker installed</span>
                  </div>
                  <div className="flex items-center gap-2 text-sm">
                    {validationResult.resource_check.docker_running ? (
                      <CheckCircle className="w-4 h-4 text-green-600" />
                    ) : (
                      <XCircle className="w-4 h-4 text-red-600" />
                    )}
                    <span>Docker daemon running</span>
                  </div>
                </div>
              </div>
            )}

            {/* Errors */}
            {validationResult.errors && validationResult.errors.length > 0 && (
              <div className="space-y-1">
                <div className="text-sm font-medium text-red-900">Errors</div>
                {validationResult.errors.map((error, i) => (
                  <div key={i} className="flex items-start gap-2 text-sm text-red-700">
                    <XCircle className="w-4 h-4 mt-0.5" />
                    <span>{error}</span>
                  </div>
                ))}
              </div>
            )}

            {/* Warnings */}
            {validationResult.warnings && validationResult.warnings.length > 0 && (
              <div className="space-y-1">
                <div className="text-sm font-medium text-yellow-900">Warnings</div>
                {validationResult.warnings.map((warning, i) => (
                  <div key={i} className="flex items-start gap-2 text-sm text-yellow-700">
                    <AlertCircle className="w-4 h-4 mt-0.5" />
                    <span>{warning}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    )
  }

  const getStatusDisplay = (status: Deployment['status']) => {
    switch (status) {
      case 'validating':
        return { label: 'Validating...', icon: <Loader2 className="w-5 h-5 animate-spin text-blue-600" />, color: 'blue' }
      case 'preparing':
        return { label: 'Preparing...', icon: <Loader2 className="w-5 h-5 animate-spin text-blue-600" />, color: 'blue' }
      case 'deploying':
        return { label: 'Deploying containers...', icon: <Loader2 className="w-5 h-5 animate-spin text-blue-600" />, color: 'blue' }
      case 'configuring':
        return { label: 'Configuring...', icon: <Loader2 className="w-5 h-5 animate-spin text-blue-600" />, color: 'blue' }
      case 'health_check':
        return { label: 'Running health checks...', icon: <Loader2 className="w-5 h-5 animate-spin text-blue-600" />, color: 'blue' }
      case 'running':
        return { label: 'Deployment Complete!', icon: <CheckCircle className="w-5 h-5 text-green-600" />, color: 'green' }
      case 'failed':
        return { label: 'Deployment Failed', icon: <XCircle className="w-5 h-5 text-red-600" />, color: 'red' }
      default:
        return { label: status, icon: <Loader2 className="w-5 h-5 animate-spin text-gray-600" />, color: 'gray' }
    }
  }

  const renderDeploy = () => {
    if (!deployment && !createDeployment.isPending) {
      // Ready to deploy
      return (
        <div className="space-y-4">
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-6">
            <div className="flex items-start gap-3">
              <CheckCircle className="w-6 h-6 text-blue-600 mt-1" />
              <div>
                <h3 className="font-medium text-blue-900 mb-2">Ready to Deploy</h3>
                <p className="text-sm text-blue-700 mb-3">
                  Click the button below to deploy {recipe.name} to {selectedDevice?.name}
                </p>
                <Button onClick={handleDeploy} disabled={createDeployment.isPending}>
                  {createDeployment.isPending ? (
                    <>
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                      Starting Deployment...
                    </>
                  ) : (
                    'Deploy Now'
                  )}
                </Button>
              </div>
            </div>
          </div>
        </div>
      )
    }

    if (deployment) {
      const statusDisplay = getStatusDisplay(deployment.status)
      const isComplete = deployment.status === 'running'
      const isFailed = deployment.status === 'failed'

      return (
        <div className="space-y-4">
          {/* Status Display */}
          <div
            className={`rounded-lg p-6 border ${
              isComplete
                ? 'bg-green-50 border-green-200'
                : isFailed
                ? 'bg-red-50 border-red-200'
                : 'bg-blue-50 border-blue-200'
            }`}
          >
            <div className="flex items-start gap-3">
              {statusDisplay.icon}
              <div className="flex-1">
                <h3
                  className={`font-medium mb-2 ${
                    isComplete
                      ? 'text-green-900'
                      : isFailed
                      ? 'text-red-900'
                      : 'text-blue-900'
                  }`}
                >
                  {statusDisplay.label}
                </h3>
                <p
                  className={`text-sm ${
                    isComplete
                      ? 'text-green-700'
                      : isFailed
                      ? 'text-red-700'
                      : 'text-blue-700'
                  }`}
                >
                  {isComplete
                    ? `${recipe.name} has been successfully deployed to ${selectedDevice?.name}`
                    : isFailed
                    ? 'The deployment encountered an error'
                    : `Deploying ${recipe.name} to ${selectedDevice?.name}...`}
                </p>
              </div>
            </div>

            {/* Deployment Logs */}
            {deployment.deployment_logs && (
              <div className="mt-4">
                <h4 className="text-sm font-medium mb-2">Deployment Logs</h4>
                <LogViewer logs={deployment.deployment_logs} />
              </div>
            )}

            {/* Error Details */}
            {isFailed && deployment.error_details && (
              <div className="mt-4 p-3 bg-red-100 rounded text-sm text-red-800">
                <strong>Error:</strong> {deployment.error_details}
              </div>
            )}

            {/* Post-deploy instructions */}
            {isComplete && recipe.post_deploy_instructions && (
              <div className="mt-4 p-4 bg-white rounded-lg border border-green-200">
                <h4 className="font-medium text-green-900 mb-2">Next Steps</h4>
                <div className="text-sm text-gray-700 whitespace-pre-wrap">
                  {recipe.post_deploy_instructions}
                </div>
              </div>
            )}
          </div>

          {/* Action Buttons */}
          {isComplete && (
            <div className="flex gap-2">
              <Button onClick={handleClose}>Close</Button>
            </div>
          )}
        </div>
      )
    }

    return null
  }

  const renderDependencies = () => (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        {dependenciesLoading
          ? 'Checking what needs to be set up...'
          : dependencies?.satisfied
          ? 'All dependencies are already satisfied!'
          : "We'll automatically set these up for you"}
      </p>

      {dependenciesLoading && (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="w-6 h-6 animate-spin text-muted-foreground" />
        </div>
      )}

      {dependenciesError && (
        <div className="p-4 bg-destructive/10 rounded-lg border border-border space-y-3">
          <div className="flex items-start gap-2">
            <AlertCircle className="w-5 h-5 text-destructive flex-shrink-0 mt-0.5" />
            <div className="flex-1">
              <p className="text-sm text-destructive font-medium">Failed to check dependencies</p>
              <p className="text-sm text-destructive/80 mt-1">{dependenciesError}</p>
            </div>
          </div>
          <Button
            size="sm"
            variant="outline"
            onClick={checkDependencies}
            disabled={dependenciesLoading}
            className="w-full"
          >
            {dependenciesLoading ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Retrying...
              </>
            ) : (
              'Retry'
            )}
          </Button>
        </div>
      )}

      {!dependenciesLoading && !dependenciesError && dependencies && (
        <>
          {dependencies.satisfied ? (
            <div className="flex items-center gap-2 p-4 bg-green-500/10 text-green-700 dark:text-green-400 rounded-lg border-border">
              <Check className="w-5 h-5 flex-shrink-0" />
              <p className="text-sm">
                All dependencies are satisfied. You're ready to deploy!
              </p>
            </div>
          ) : (
            <div className="space-y-4">
              {dependencies.to_provision.map((dep, index) => (
                <div
                  key={index}
                  className="p-4 border border-border rounded-lg space-y-2 bg-card"
                >
                  <div className="flex items-start gap-3">
                    <div className="text-2xl">{getDependencyIcon(dep.type)}</div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <h4 className="font-medium capitalize">
                          {getDependencyName(dep)}
                        </h4>
                        {dep.engine && (
                          <span className="text-xs px-2 py-0.5 rounded-full bg-secondary/20 text-secondary-foreground">
                            {dep.engine}
                          </span>
                        )}
                      </div>
                      <p className="text-sm text-muted-foreground mt-1">
                        {dep.purpose || dep.message || `Provides ${dep.type.replace('_', ' ')} functionality`}
                      </p>
                      <div className="flex items-center gap-4 mt-2 text-xs text-muted-foreground">
                        {dep.estimated_time_seconds && (
                          <div className="flex items-center gap-1">
                            <Clock className="w-3 h-3" />
                            Est. {formatTime(dep.estimated_time_seconds)}
                          </div>
                        )}
                        {dep.estimated_ram_mb && (
                          <div className="flex items-center gap-1">
                            <Cpu className="w-3 h-3" />
                            {formatRAM(dep.estimated_ram_mb)} RAM
                          </div>
                        )}
                        {dep.estimated_storage_gb && (
                          <div className="flex items-center gap-1">
                            <HardDrive className="w-3 h-3" />
                            {dep.estimated_storage_gb}GB storage
                          </div>
                        )}
                      </div>
                    </div>
                    <span className="text-xs px-2 py-1 rounded-md bg-muted text-muted-foreground flex-shrink-0">
                      Auto-deploy
                    </span>
                  </div>
                </div>
              ))}

              {/* Total Summary */}
              {dependencies.to_provision.length > 0 && (
                <div className="p-4 bg-muted/50 rounded-lg border border-border">
                  <h4 className="font-medium mb-2">Total Requirements</h4>
                  <div className="flex items-center gap-4 text-sm">
                    {dependencies.total_estimated_time_seconds > 0 && (
                      <div className="flex items-center gap-1">
                        <Clock className="w-4 h-4" />
                        ~{formatTime(dependencies.total_estimated_time_seconds)}
                      </div>
                    )}
                    {dependencies.total_estimated_ram_mb > 0 && (
                      <div className="flex items-center gap-1">
                        <Cpu className="w-4 h-4" />
                        {formatRAM(dependencies.total_estimated_ram_mb)} RAM
                      </div>
                    )}
                    {dependencies.total_estimated_storage_gb > 0 && (
                      <div className="flex items-center gap-1">
                        <HardDrive className="w-4 h-4" />
                        {dependencies.total_estimated_storage_gb}GB storage
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          )}
        </>
      )}
    </div>
  )

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Deploy {recipe.name}</DialogTitle>
          <DialogDescription>{recipe.tagline}</DialogDescription>
        </DialogHeader>

        {renderStepIndicator()}

        <div className="py-4">
          {currentStep === 'select-device' && renderSelectDevice()}
          {currentStep === 'dependencies' && renderDependencies()}
          {currentStep === 'configure' && renderConfigure()}
          {currentStep === 'validate' && renderValidate()}
          {currentStep === 'deploy' && renderDeploy()}
        </div>

        <DialogFooter>
          <div className="flex justify-between w-full">
            <Button variant="outline" onClick={handleBack} disabled={currentStep === 'select-device'}>
              Back
            </Button>
            <div className="flex gap-2">
              <Button variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              {currentStep !== 'deploy' && (
                <Button
                  onClick={handleNext}
                  disabled={
                    (currentStep === 'select-device' && (!selectedDeviceId || dependenciesLoading)) ||
                    (currentStep === 'dependencies' && (dependenciesLoading || !!dependenciesError)) ||
                    (currentStep === 'validate' && validateDeployment.isPending) ||
                    (currentStep === 'validate' && !validateDeployment.data?.valid)
                  }
                >
                  {dependenciesLoading ? (
                    <>
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                      Checking...
                    </>
                  ) : currentStep === 'validate' ? (
                    'Continue'
                  ) : (
                    'Next'
                  )}
                </Button>
              )}
            </div>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
