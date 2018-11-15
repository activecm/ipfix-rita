#!/usr/bin/env python3

import subprocess
import sys
import os.path
import os
import signal
import threading
import tempfile
import time
import shutil


"""
Built against:
tcpdump --version:
    tcpdump version 4.9.2
    libpcap version 1.8.1
    OpenSSL 1.0.2g  1 Mar 2016

docker-compose --version
    docker-compose version 1.22.0, build f46880fe
"""


def OrEvent(*events):
    """
    Used to wait for any one of a set of given events.
    """
    or_event = threading.Event()

    def or_set(self):
        self._set()
        self.changed()

    def or_clear(self):
        self._clear()
        self.changed()

    def changed():
        if any([e.is_set() for e in events]):
            or_event.set()
        else:
            or_event.clear()

    for e in events:
        e._set = e.set
        e._clear = e.clear
        e.changed = changed
        e.set = lambda e=e: or_set(e)
        e.clear = lambda e=e: or_clear(e)
    changed()
    return or_event


class StoppedException(Exception):

    """ Indicates a thread was stopped """
    pass


class TCPDump_Monitor(threading.Thread):

    """
    Uses TCPDump to monitor for netflow/ ipfix data. Fires self.data_found
    when data is seen crossing the wire. May fire exception_encountered
    if a problem arises. Once self.data_found has been fired, the first
    instance of tcpdump is killed off, and a second one is started which
    writes to file.
    """

    def __init__(self, pcap_file_path, ipfix_iface):
        threading.Thread.__init__(self)
        self.data_found = threading.Event()
        self.exception_encountered = threading.Event()
        self.__stop_event = threading.Event()
        self.pcap_file_path = pcap_file_path
        self.ipfix_iface = ipfix_iface
        self.tcpdump_proc = None

    def stop(self):
        # The process gets killed here in case
        # the run method is blocking on a piped read
        if self.tcpdump_proc is not None and self.tcpdump_proc.poll() is None:
            os.killpg(os.getpgid(self.tcpdump_proc.pid), signal.SIGTERM)
        self.__stop_event.set()

    def stopped(self):
        return self.__stop_event.is_set()

    def run(self):
        try:
            self.tcpdump_proc = subprocess.Popen(
                ["tcpdump", "-l", "-i", self.ipfix_iface, "udp port 2055"],
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                start_new_session=True
            )

            for line in iter(self.tcpdump_proc.stdout.readline, b''):
                # iter(tcpdump_proc.stdout.readline, b'') should exit
                # when the process is killed
                line = line.decode(sys.stdout.encoding)

                if "UDP" in line and not self.data_found.is_set():
                    self.data_found.set()
                    break

            if self.stopped():
                raise StoppedException()

            os.killpg(os.getpgid(self.tcpdump_proc.pid), signal.SIGTERM)
            for line in iter(self.tcpdump_proc.stdout.readline, b''):
                # line = line.decode(sys.stdout.encoding)
                # print(line, end="")
                pass
            self.tcpdump_proc.wait()

            if self.stopped():
                raise StoppedException()

            self.tcpdump_proc = subprocess.Popen(
                ["tcpdump", "-i", self.ipfix_iface, "-C", "50", "-w",
                    self.pcap_file_path, "-s", "0", "udp port 2055"],
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                start_new_session=True
            )

            self.__stop_event.wait()

        except (ValueError, OSError):
            print("Unable to start tcpdump.")
            self.exception_encountered.set()
        except StoppedException:
            pass
            # Stop was called before we were ready to stop
            # No exception was encountered as this is defined behavior.
        except Exception as e:
            print("An error occurred while running tcpdump.")
            print(e)
            self.exception_encountered.set()

        finally:
            if not self.stopped():
                self.stop()

            if self.tcpdump_proc is not None and self.tcpdump_proc.poll() is None:
                # You must drain a process' pipe before you can .wait() for it
                for line in iter(self.tcpdump_proc.stdout.readline, b''):
                    # line = line.decode(sys.stdout.encoding)
                    # print(line, end="")
                    pass
                self.tcpdump_proc.wait()


class IPFIX_RITA_Monitor(threading.Thread):

    """
    Monitors the ipfix-rita application log for errors and records
    a copy of the log. Fires self.error_found once an error has been found
    in the logs. May fire self.exception_encountered if a problem is encountered
    during the process.
    """

    IPFIX_RITA = "ipfix-rita"

    def __init__(self, log_file_path):
        threading.Thread.__init__(self)
        self.error_found = threading.Event()
        self.exception_encountered = threading.Event()
        self.__stop_event = threading.Event()
        self.__docker_compose_attached_event = threading.Event()
        self.log_file_path = log_file_path
        self.ipfix_rita_proc = None

    def stop(self):
        # The process gets killed here in case
        # the run method is blocking on a piped read
        if self.ipfix_rita_proc is not None and self.ipfix_rita_proc.poll() is None:
            # Have to wait for the attach step from docker-compose
            # otherwise, docker-compose doesn't actually stop everything.
            self.__docker_compose_attached_event.wait()
            os.killpg(os.getpgid(self.ipfix_rita_proc.pid), signal.SIGTERM)
        self.__stop_event.set()

    def stopped(self):
        return self.__stop_event.is_set()

    def run(self):
        try:
            log_file = open(self.log_file_path, 'w')
        except:
            print("Could not open {0}".format(self.log_file_path))
            self.exception_encountered.set()
            return

        try:
            subprocess.check_call(
                [IPFIX_RITA_Monitor.IPFIX_RITA, "stop"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL
            )

            if self.stopped():
                raise StoppedException()

            self.ipfix_rita_proc = subprocess.Popen(
                [IPFIX_RITA_Monitor.IPFIX_RITA, "up"],
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                start_new_session=True
            )

            for line in iter(self.ipfix_rita_proc.stdout.readline, b''):
                # iter(ipfix_rita_proc.stdout.readline, b'') should exit
                # when the process is killed
                line = line.decode(sys.stdout.encoding)
                # print(line, end="")
                if "ERR" in line and not self.error_found.is_set():
                    self.error_found.set()
                if "Attaching" in line and not self.__docker_compose_attached_event.is_set():
                    self.__docker_compose_attached_event.set()
                log_file.write(line)

            self.__stop_event.wait()
            self.ipfix_rita_proc.wait()

        except (ValueError, OSError):
            print("Could not execute IPFIX-RITA.")
            self.exception_encountered.set()
        except StoppedException:
            pass
            # Stop was called before we were ready to stop
            # No exception was encountered as this is defined behavior.
        except Exception as e:
            print("An error occurred while monitoring ipfix-rita.")
            print(e)
            self.exception_encountered.set()
        finally:
            if not self.stopped():
                self.stop()

            log_file.close()

LOG_FILE_NAME = "ipfix-rita-log.txt"
PCAP_FILE_NAME = "ipfix-data.pcap"
ARCHIVE_NAME = "ipfix-rita-debug"
WAIT_MINUTES = 5  # How long to wait for the detection of ipfix data/ errors
MONITOR_MINUTES = 5  # How long to record ipfix data/ the error log


def main():
    print(
        "This script will aid you in collecting diagnostic information from IPFIX-RITA."
    )
    print(
        "First, the script will ensure that traffic is being received on UDP port 2055 and begin a packet capture."
    )
    print(
        "Next, the script will restart ipfix-rita, record the application log, and look for errors."
    )
    print("")

    if os.geteuid() != 0:
        print("This script requires administrator privileges.")
        return 1

    if shutil.which("ipfix-rita") is None:
        print("IPFIX-RITA is not installed.")
        return 1

    if shutil.which("tcpdump") is None:
        print("tcpdump is not installed")
        self.exception_encountered.set()
        return 1

    ipfix_iface = input(
        "Which interface is being used to collect Netflow/ IPFIX data? "
    )
    print("")

    with tempfile.TemporaryDirectory() as tmp_dir_path:
        try:
            archive_folder = os.path.join(tmp_dir_path, ARCHIVE_NAME)
            os.mkdir(archive_folder, 0o0755)
            pcap_file_path = os.path.join(archive_folder, PCAP_FILE_NAME)
            tcpdump_monitor = TCPDump_Monitor(pcap_file_path, ipfix_iface)
            print(
                "Monitoring {0} for UDP data on port 2055...".format(
                    ipfix_iface
                )
            )
            print(
                "The monitor will wait up to {0} minutes for data to arrive.".format(
                    WAIT_MINUTES
                )
            )
            tcpdump_data_or_exception_event = OrEvent(
                tcpdump_monitor.data_found,
                tcpdump_monitor.exception_encountered,
            )
            tcpdump_monitor.start()
            if not tcpdump_data_or_exception_event.wait(timeout=60 * WAIT_MINUTES):
                print(
                    "No UDP data was found on port 2055 on the interface {0}.".format(
                        ipfix_iface
                    )
                )
                print(
                    "The script can not continue without IPFIX/ Netflow v9 data."
                )
                tcpdump_monitor.stop()
                tcpdump_monitor.join()
                return 1
            elif tcpdump_monitor.exception_encountered.is_set():
                tcpdump_monitor.stop()
                tcpdump_monitor.join()
                return 1

            assert tcpdump_monitor.data_found.is_set()
        except (KeyboardInterrupt, SystemExit):
            print("Stopping the IPFIX-RITA debug script...")
            tcpdump_monitor.stop()
            tcpdump_monitor.join()
            return 1
        except Exception as e:
            print(e)
            tcpdump_monitor.stop()
            tcpdump_monitor.join()
            return 1

        print("UDP data was found on port 2055 on {0}.".format(ipfix_iface))
        print(
            "TCPDump will begin recording UDP data on port 2055 on the interface {0}.".format(
                ipfix_iface
            )
        )
        print("")
        try:
            log_file_path = os.path.join(archive_folder, LOG_FILE_NAME)
            ipfix_rita_monitor = IPFIX_RITA_Monitor(log_file_path)
            print(
                "Restarting IPFIX-RITA and monitoring the application log for errors..."
            )
            print(
                "The script will wait for up to {0} minutes for an error to occur.".format(
                    WAIT_MINUTES
                )
            )
            ipfixErrOrExceptionEvent = OrEvent(
                ipfix_rita_monitor.error_found,
                ipfix_rita_monitor.exception_encountered
            )
            ipfix_rita_monitor.start()
            if not ipfixErrOrExceptionEvent.wait(timeout=WAIT_MINUTES * 60):
                print("No errors were found! Please check if RITA has received any data by")
                print("running `rita show-databases` and looking for new results.")
                print("If RITA has successfully received data from IPFIX-RITA and your")
                print("IPFIX/ Netflow v9 device is not listed under the compatibility matrix")
                print("in the main README.md, please contact support@activecountermeasures.com,")
                print("and we will add your device to the list.")
                print("")
                print("Thank you for your time.")
                tcpdump_monitor.stop()
                tcpdump_monitor.join()
                ipfix_rita_monitor.stop()
                ipfix_rita_monitor.join()
                return 0
            elif ipfix_rita_monitor.exception_encountered.is_set():
                tcpdump_monitor.stop()
                tcpdump_monitor.join()
                ipfix_rita_monitor.stop()
                ipfix_rita_monitor.join()
                return 1

            assert ipfix_rita_monitor.error_found.is_set()

            print("An error was found in the IPFIX-RITA logs.")
            print(
                "Monitoring IPFIX-RITA for more errors for {0} minutes...".format(
                    MONITOR_MINUTES
                )
            )
            time.sleep(MONITOR_MINUTES * 60)
        except (KeyboardInterrupt, SystemExit):
            print("Stopping the IPFIX-RITA debug script...")
            # The finally will execute before the return
            return 1
        except Exception as e:
            print(e)
            # the finally will execute before the return
            return 1
        finally:
            tcpdump_monitor.stop()
            tcpdump_monitor.join()
            ipfix_rita_monitor.stop()
            ipfix_rita_monitor.join()

        print("")
        print("TCPDump and IPFIX-RITA have been stopped.")
        print("The script will now package the results.")

        try:
            subprocess.check_call(
                ["tar", "-C", tmp_dir_path, "-czf", "{0}.tgz".format(ARCHIVE_NAME), ARCHIVE_NAME]
            )
        except subprocess.CalledProcessError:
            print(
                "Could not create tarball containing the collected debug files"
            )
            return 1

    print(
        "Please email {0}.tgz along with a description of the settings and model of your IPFIX/ Netflow v9".format(
            ARCHIVE_NAME
        )
    )
    print(
        "device to support@activecountermeasures.com for further assistance."
    )
    print("")
    print(
        "Thank you for your time."
    )
    return 0

if __name__ == "__main__":
    main()
