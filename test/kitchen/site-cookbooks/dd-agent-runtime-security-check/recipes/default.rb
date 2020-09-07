#
# Cookbook Name:: dd-agent-runtime-security-check
# Recipe:: default
#
# Copyright (C) 2020 Datadog
#

if node['platform_family'] != 'windows'
  wrk_dir = '/tmp/runtime-security'

  directory wrk_dir do
    recursive true
  end

  cookbook_file "#{wrk_dir}/testsuite" do
    source "testsuite"
    mode '755'
  end
end