require 'spec_helper'

if os != :windows
  describe command('sudo /tmp/runtime-security/testsuite -test.run ^TestMkdir$') do
    its('exit_status') { should eq 0 }
  end
end