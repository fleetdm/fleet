package oracle

import (
	"reflect"
	"testing"

	"github.com/vulsio/goval-dictionary/models"
)

func Test_collectOraclePacks(t *testing.T) {
	type args struct {
		cri Criteria
	}
	tests := []struct {
		name string
		args args
		want []distroPackage
	}{
		{
			name: "single ver and single arch",
			args: args{
				cri: Criteria{
					Operator:   "AND",
					Criterions: []Criterion{{Comment: "Oracle Linux 8 is installed"}},
					Criterias: []Criteria{
						{
							Operator:   "AND",
							Criterions: []Criterion{{Comment: "Oracle Linux arch is x86_64"}},
							Criterias: []Criteria{
								{
									Operator: "OR",
									Criterias: []Criteria{
										{
											Operator: "AND",
											Criterions: []Criterion{
												{Comment: "kernel-uek-container is earlier than 0:5.4.17-2136.324.5.3.el8"},
												{Comment: "kernel-uek-container is signed with the Oracle Linux 8 key"},
											},
										},
										{
											Operator: "AND",
											Criterions: []Criterion{
												{Comment: "kernel-uek-container-debug is earlier than 0:5.4.17-2136.324.5.3.el8"},
												{Comment: "kernel-uek-container-debug is signed with the Oracle Linux 8 key"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: []distroPackage{
				{
					osVer: "8",
					pack: models.Package{
						Name:    "kernel-uek-container",
						Version: "0:5.4.17-2136.324.5.3.el8",
						Arch:    "x86_64",
					},
				},
				{
					osVer: "8",
					pack: models.Package{
						Name:    "kernel-uek-container-debug",
						Version: "0:5.4.17-2136.324.5.3.el8",
						Arch:    "x86_64",
					},
				},
			},
		},
		{
			name: "single ver and multiple arch",
			args: args{
				cri: Criteria{
					Operator:   "AND",
					Criterions: []Criterion{{Comment: "Oracle Linux 9 is installed"}},
					Criterias: []Criteria{
						{
							Operator: "OR",
							Criterias: []Criteria{
								{
									Operator:   "AND",
									Criterions: []Criterion{{Comment: "Oracle Linux arch is aarch64"}},
									Criterias: []Criteria{
										{
											Operator: "OR",
											Criterias: []Criteria{
												{
													Operator: "AND",
													Criterions: []Criterion{
														{Comment: "osbuild is earlier than 0:81-1.el9"},
														{Comment: "osbuild is signed with the Oracle Linux 9 key"},
													},
												},
											},
										},
									},
								},
								{
									Operator:   "AND",
									Criterions: []Criterion{{Comment: "Oracle Linux arch is x86_64"}},
									Criterias: []Criteria{
										{
											Operator: "OR",
											Criterias: []Criteria{
												{
													Operator: "AND",
													Criterions: []Criterion{
														{Comment: "osbuild is earlier than 0:81-1.el9"},
														{Comment: "osbuild is signed with the Oracle Linux 9 key"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: []distroPackage{
				{
					osVer: "9",
					pack: models.Package{
						Name:    "osbuild",
						Version: "0:81-1.el9",
						Arch:    "aarch64",
					},
				},
				{
					osVer: "9",
					pack: models.Package{
						Name:    "osbuild",
						Version: "0:81-1.el9",
						Arch:    "x86_64",
					},
				},
			},
		},
		{
			name: "multiple ver and single arch",
			args: args{
				cri: Criteria{
					Operator: "OR",
					Criterias: []Criteria{
						{
							Operator:   "AND",
							Criterions: []Criterion{{Comment: "Oracle Linux 6 is installed"}},
							Criterias: []Criteria{
								{
									Operator:   "AND",
									Criterions: []Criterion{{Comment: "Oracle Linux arch is x86_64"}},
									Criterias: []Criteria{
										{
											Operator: "OR",
											Criterias: []Criteria{
												{
													Operator: "AND",
													Criterions: []Criterion{
														{Comment: "kernel-uek is earlier than 0:3.8.13-118.17.4.el6uek"},
														{Comment: "kernel-uek is signed with the Oracle Linux 6 key"},
													},
												},
												{
													Operator: "AND",
													Criterions: []Criterion{
														{Comment: "kernel-uek-debug is earlier than 0:3.8.13-118.17.4.el6uek"},
														{Comment: "kernel-uek-debug is signed with the Oracle Linux 6 key"},
													},
												},
											},
										},
									},
								},
							},
						},
						{
							Operator:   "AND",
							Criterions: []Criterion{{Comment: "Oracle Linux 7 is installed"}},
							Criterias: []Criteria{
								{
									Operator:   "AND",
									Criterions: []Criterion{{Comment: "Oracle Linux arch is x86_64"}},
									Criterias: []Criteria{
										{
											Operator: "OR",
											Criterias: []Criteria{
												{
													Operator: "AND",
													Criterions: []Criterion{
														{Comment: "kernel-uek is earlier than 0:3.8.13-118.17.4.el7uek"},
														{Comment: "kernel-uek is signed with the Oracle Linux 7 key"},
													},
												},
												{
													Operator: "AND",
													Criterions: []Criterion{
														{Comment: "kernel-uek-debug is earlier than 0:3.8.13-118.17.4.el7uek"},
														{Comment: "kernel-uek-debug is signed with the Oracle Linux 7 key"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: []distroPackage{
				{
					osVer: "6",
					pack: models.Package{
						Name:    "kernel-uek",
						Version: "0:3.8.13-118.17.4.el6uek",
						Arch:    "x86_64",
					},
				},
				{
					osVer: "6",
					pack: models.Package{
						Name:    "kernel-uek-debug",
						Version: "0:3.8.13-118.17.4.el6uek",
						Arch:    "x86_64",
					},
				},
				{
					osVer: "7",
					pack: models.Package{
						Name:    "kernel-uek",
						Version: "0:3.8.13-118.17.4.el7uek",
						Arch:    "x86_64",
					},
				},
				{
					osVer: "7",
					pack: models.Package{
						Name:    "kernel-uek-debug",
						Version: "0:3.8.13-118.17.4.el7uek",
						Arch:    "x86_64",
					},
				},
			},
		},
		{
			name: "multiple ver and multiple arch",
			args: args{
				cri: Criteria{
					Operator: "OR",
					Criterias: []Criteria{
						{
							Operator:   "AND",
							Criterions: []Criterion{{Comment: "Oracle Linux 5 is installed"}},
							Criterias: []Criteria{
								{
									Operator: "OR",
									Criterias: []Criteria{
										{
											Operator:   "AND",
											Criterions: []Criterion{{Comment: "Oracle Linux arch is x86_64"}},
											Criterias: []Criteria{
												{
													Operator: "OR",
													Criterias: []Criteria{
														{
															Operator: "AND",
															Criterions: []Criterion{
																{Comment: "kernel-uek is earlier than 0:2.6.39-400.294.6.el5uek"},
																{Comment: "kernel-uek is signed with the Oracle Linux 5 key"},
															},
														},
													},
												},
											},
										},
									},
								},
								{
									Operator: "OR",
									Criterias: []Criteria{
										{
											Operator:   "AND",
											Criterions: []Criterion{{Comment: "Oracle Linux arch is i386"}},
											Criterias: []Criteria{
												{
													Operator: "OR",
													Criterias: []Criteria{
														{
															Operator: "AND",
															Criterions: []Criterion{
																{Comment: "kernel-uek is earlier than 0:2.6.39-400.294.6.el5uek"},
																{Comment: "kernel-uek is signed with the Oracle Linux 5 key"},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
						{
							Operator:   "AND",
							Criterions: []Criterion{{Comment: "Oracle Linux 6 is installed"}},
							Criterias: []Criteria{
								{
									Operator: "OR",
									Criterias: []Criteria{
										{
											Operator:   "AND",
											Criterions: []Criterion{{Comment: "Oracle Linux arch is x86_64"}},
											Criterias: []Criteria{
												{
													Operator: "OR",
													Criterias: []Criteria{
														{
															Operator: "AND",
															Criterions: []Criterion{
																{Comment: "kernel-uek is earlier than 0:2.6.39-400.294.6.el6uek"},
																{Comment: "kernel-uek is signed with the Oracle Linux 6 key"},
															},
														},
													},
												},
											},
										},
									},
								},
								{
									Operator: "OR",
									Criterias: []Criteria{
										{
											Operator:   "AND",
											Criterions: []Criterion{{Comment: "Oracle Linux arch is i386"}},
											Criterias: []Criteria{
												{
													Operator: "OR",
													Criterias: []Criteria{
														{
															Operator: "AND",
															Criterions: []Criterion{
																{Comment: "kernel-uek is earlier than 0:2.6.39-400.294.6.el6uek"},
																{Comment: "kernel-uek is signed with the Oracle Linux 6 key"},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: []distroPackage{
				{
					osVer: "5",
					pack: models.Package{
						Name:    "kernel-uek",
						Version: "0:2.6.39-400.294.6.el5uek",
						Arch:    "x86_64",
					},
				},
				{
					osVer: "5",
					pack: models.Package{
						Name:    "kernel-uek",
						Version: "0:2.6.39-400.294.6.el5uek",
						Arch:    "i386",
					},
				},
				{
					osVer: "6",
					pack: models.Package{
						Name:    "kernel-uek",
						Version: "0:2.6.39-400.294.6.el6uek",
						Arch:    "x86_64",
					},
				},
				{
					osVer: "6",
					pack: models.Package{
						Name:    "kernel-uek",
						Version: "0:2.6.39-400.294.6.el6uek",
						Arch:    "i386",
					},
				},
			},
		},
		{
			name: "single modularitylabel",
			args: args{
				cri: Criteria{
					Operator:   "AND",
					Criterions: []Criterion{{Comment: "Oracle Linux 8 is installed"}},
					Criterias: []Criteria{
						{
							Operator: "OR",
							Criterias: []Criteria{
								{
									Operator:   "AND",
									Criterions: []Criterion{{Comment: "Oracle Linux arch is aarch64"}},
									Criterias: []Criteria{
										{
											Operator:   "AND",
											Criterions: []Criterion{{Comment: "Module container-tools:ol8 is enabled"}},
											Criterias: []Criteria{
												{
													Operator: "OR",
													Criterias: []Criteria{
														{
															Criterions: []Criterion{
																{Comment: "runc is earlier than 0:1.0.0-55.rc5.dev.git2abd837.module+el8.0.0+5215+77f672ad"},
																{Comment: "runc is signed with the Oracle Linux 8 key"},
															},
														},
													},
												},
											},
										},
									},
								},
								{
									Operator:   "AND",
									Criterions: []Criterion{{Comment: "Oracle Linux arch is x86_64"}},
									Criterias: []Criteria{
										{
											Operator:   "AND",
											Criterions: []Criterion{{Comment: "Module container-tools:ol8 is enabled"}},
											Criterias: []Criteria{
												{
													Operator: "OR",
													Criterias: []Criteria{
														{
															Criterions: []Criterion{
																{Comment: "runc is earlier than 0:1.0.0-55.rc5.dev.git2abd837.module+el8.0.0+5215+77f672ad"},
																{Comment: "runc is signed with the Oracle Linux 8 key"},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: []distroPackage{
				{
					osVer: "8",
					pack: models.Package{
						Name:            "runc",
						Version:         "0:1.0.0-55.rc5.dev.git2abd837.module+el8.0.0+5215+77f672ad",
						Arch:            "aarch64",
						ModularityLabel: "container-tools:ol8",
					},
				},
				{
					osVer: "8",
					pack: models.Package{
						Name:            "runc",
						Version:         "0:1.0.0-55.rc5.dev.git2abd837.module+el8.0.0+5215+77f672ad",
						Arch:            "x86_64",
						ModularityLabel: "container-tools:ol8",
					},
				},
			},
		},
		{
			name: "multiple modularitylabel",
			args: args{
				cri: Criteria{
					Operator:   "AND",
					Criterions: []Criterion{{Comment: "Oracle Linux 8 is installed"}},
					Criterias: []Criteria{
						{
							Operator: "OR",
							Criterias: []Criteria{
								{
									Operator:   "AND",
									Criterions: []Criterion{{Comment: "Oracle Linux arch is x86_64"}},
									Criterias: []Criteria{
										{
											Operator: "OR",
											Criterias: []Criteria{
												{
													Operator:   "AND",
													Criterions: []Criterion{{Comment: "Module virt:ol is enabled"}},
													Criterias: []Criteria{
														{
															Operator: "OR",
															Criterias: []Criteria{
																{
																	Operator: "AND",
																	Criterions: []Criterion{
																		{Comment: "libvirt is earlier than 0:4.5.0-35.0.1.module+el8.1.0+5378+c5e0f4d7"},
																		{Comment: "libvirt is signed with the Oracle Linux 8 key"},
																	},
																},
															},
														},
													},
												},
												{
													Operator:   "AND",
													Criterions: []Criterion{{Comment: "Module virt-devel:ol is enabled"}},
													Criterias: []Criteria{
														{
															Operator: "AND",
															Criterions: []Criterion{
																{Comment: "qemu-kvm-tests is earlier than 15:2.12.0-88.0.1.module+el8.1.0+5378+c5e0f4d7"},
																{Comment: "qemu-kvm-tests is signed with the Oracle Linux 8 key"},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: []distroPackage{
				{
					osVer: "8",
					pack: models.Package{
						Name:            "libvirt",
						Version:         "0:4.5.0-35.0.1.module+el8.1.0+5378+c5e0f4d7",
						Arch:            "x86_64",
						ModularityLabel: "virt:ol",
					},
				},
				{
					osVer: "8",
					pack: models.Package{
						Name:            "qemu-kvm-tests",
						Version:         "15:2.12.0-88.0.1.module+el8.1.0+5378+c5e0f4d7",
						Arch:            "x86_64",
						ModularityLabel: "virt-devel:ol",
					},
				},
			},
		},
		{
			name: "modular package mix with not modular package",
			args: args{
				cri: Criteria{
					Operator:   "AND",
					Criterions: []Criterion{{Comment: "Oracle Linux 8 is installed"}},
					Criterias: []Criteria{
						{
							Operator: "OR",
							Criterias: []Criteria{
								{
									Operator:   "AND",
									Criterions: []Criterion{{Comment: "Oracle Linux arch is x86_64"}},
									Criterias: []Criteria{
										{
											Operator: "OR",
											Criterias: []Criteria{
												{
													Operator:   "AND",
													Criterions: []Criterion{{Comment: "Module name:stream is enabled"}},
													Criterias: []Criteria{
														{
															Operator: "OR",
															Criterias: []Criteria{
																{
																	Operator: "AND",
																	Criterions: []Criterion{
																		{Comment: "package is earlier than 0:0.0.1.module+el8.1.0+5378+c5e0f4d7"},
																		{Comment: "package is signed with the Oracle Linux 8 key"},
																	},
																},
															},
														},
													},
												},
												{
													Operator: "AND",
													Criterias: []Criteria{
														{
															Operator: "OR",
															Criterias: []Criteria{
																{
																	Operator: "AND",
																	Criterions: []Criterion{
																		{Comment: "package is earlier than 0.0.1.el8"},
																		{Comment: "package is signed with the Oracle Linux 8 key"},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: []distroPackage{
				{
					osVer: "8",
					pack: models.Package{
						Name:            "package",
						Version:         "0:0.0.1.module+el8.1.0+5378+c5e0f4d7",
						Arch:            "x86_64",
						ModularityLabel: "name:stream",
					},
				},
				{
					osVer: "8",
					pack: models.Package{
						Name:    "package",
						Version: "0.0.1.el8",
						Arch:    "x86_64",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := collectOraclePacks(tt.args.cri); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("collectOraclePacks() = %v, want %v", got, tt.want)
			}
		})
	}
}
