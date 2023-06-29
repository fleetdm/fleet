#Requires -Version 5.0
param (
    [Parameter(Mandatory = $true)]
    [int]$TargetPID,

    [Parameter(Mandatory = $false)]
    [int]$SampleIntervalInSecs = 10 #default to 10 secs
)

Write-Host "PID to profile is $TargetPID."

# Path for the CSV and chart file
$currentDirectory = Get-Location
$csvFilePath = $currentDirectory.Path + "\process_profile_$TargetPID.csv"
$chartFilePathCPU = $currentDirectory.Path + "\process_profile_cpu_$TargetPID.png"
$chartFilePathMemory = $currentDirectory.Path + "\process_profile_memory_$TargetPID.png"

# Initialize an empty array for storing data
$data = @()

# Load the required .NET assembly for charting
[void][Reflection.Assembly]::LoadWithPartialName("System.Windows.Forms.DataVisualization")

# Create two charts and define CPU and Memory series
$chartCPU = New-Object System.Windows.Forms.DataVisualization.Charting.Chart
$chartCPU.Width = 600
$chartCPU.Height = 400

$chartMemory = New-Object System.Windows.Forms.DataVisualization.Charting.Chart
$chartMemory.Width = 600
$chartMemory.Height = 400

$chartAreaCPU = New-Object System.Windows.Forms.DataVisualization.Charting.ChartArea
$chartAreaMemory = New-Object System.Windows.Forms.DataVisualization.Charting.ChartArea

$chartAreaCPU.AxisY.Title = "CPU Percentage"
$chartAreaCPU.AxisX.Title = "Time in Seconds"
$chartAreaMemory.AxisY.Title = "Memory Megabytes"
$chartAreaMemory.AxisX.Title = "Time in Seconds"

$chartAreaCPU.AxisX.Minimum = 0
$chartAreaMemory.AxisX.Minimum = 0

$chartCPU.ChartAreas.Add($chartAreaCPU)
$chartMemory.ChartAreas.Add($chartAreaMemory)

$cpuSeries = New-Object System.Windows.Forms.DataVisualization.Charting.Series
$cpuSeries.Name = 'CPU'
$cpuSeries.ChartType = [System.Windows.Forms.DataVisualization.Charting.SeriesChartType]::Line

$memSeries = New-Object System.Windows.Forms.DataVisualization.Charting.Series
$memSeries.Name = 'Memory'
$memSeries.ChartType = [System.Windows.Forms.DataVisualization.Charting.SeriesChartType]::Line

$chartCPU.Series.Add($cpuSeries)
$chartMemory.Series.Add($memSeries)

# Initialize time 
$startTime = Get-Date
$cpuCores = (Get-WMIObject Win32_ComputerSystem).NumberOfLogicalProcessors

Write-Host "Press any key to stop the data collection and generate profile charts."
while($true) {
    if([System.Console]::KeyAvailable) {
        Write-Host "Key Pressed. Stopping script..."
        break
    }

    # Get the process current information
    $process = Get-Process -Id $TargetPID -ErrorAction SilentlyContinue

    if($process) {
        $processName = $process.Name 
        $cpu = [Decimal]::Round((((Get-Counter "\Process($processName*)\% Processor Time" -SampleInterval 1).CounterSamples[0].CookedValue) / $cpuCores), 2)
        $mem = $process.PagedMemorySize64 / 1MB # Convert to MB

        # Create a custom object to hold the data
        $timeInSeconds = [math]::Round(((Get-Date) - $startTime).TotalSeconds)
        $obj = New-Object PSObject
        $obj | Add-Member -MemberType NoteProperty -Name "TimeInSecs" -Value $timeInSeconds
        $obj | Add-Member -MemberType NoteProperty -Name "CPU" -Value $cpu
        $obj | Add-Member -MemberType NoteProperty -Name "Memory" -Value $mem

        # Add the object to the data array
        $data += $obj

        # Add points to the chart series
        $null = $cpuSeries.Points.AddXY($timeInSeconds, $cpu)
        $null = $memSeries.Points.AddXY($timeInSeconds, $mem)

        # Export the data to CSV
        $data | Export-Csv -Path $csvFilePath -NoTypeInformation
    } else {
        break
    }

    # Sleep for the sample interval
    Start-Sleep -Seconds $SampleIntervalInSecs
}

$chartCPU.SaveImage($chartFilePathCPU, [System.Windows.Forms.DataVisualization.Charting.ChartImageFormat]::Png)
$chartMemory.SaveImage($chartFilePathMemory, [System.Windows.Forms.DataVisualization.Charting.ChartImageFormat]::Png)
