package owl

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/printer"
	"go.uber.org/zap"
)

type QueryNodeReducer func([]*OperationSet, *ast.OperationDefinition, *ast.SelectionSet) (*ast.SelectionSet, error)

func (s *Store) snapshotQuery(query, vars io.StringWriter, resolve bool) error {
	varDefs := []*ast.VariableDefinition{
		ast.NewVariableDefinition(&ast.VariableDefinition{
			Variable: ast.NewVariable(&ast.Variable{
				Name: ast.NewName(&ast.Name{
					Value: "insecure",
				}),
			}),
			Type: ast.NewNamed(&ast.Named{
				Name: ast.NewName(&ast.Name{
					Value: "Boolean",
				}),
			}),
			DefaultValue: ast.NewBooleanValue(&ast.BooleanValue{
				Value: false,
			}),
		}),
	}

	loaded, updated, deleted := 0, 0, 0
	for _, opSet := range s.opSets {
		if len(opSet.specs) == 0 && len(opSet.values) == 0 {
			continue
		}
		switch opSet.operation.kind {
		case LoadSetOperation:
			loaded++
		case UpdateSetOperation:
			updated++
		case DeleteSetOperation:
			deleted++
		}

	}
	s.logger.Debug("snapshot opSets breakdown", zap.Int("loaded", loaded), zap.Int("updated", updated), zap.Int("deleted", deleted), zap.Int("total", len(s.opSets)))

	reducers := []QueryNodeReducer{
		reconcileAsymmetry(s),
		reduceSetOperations(vars),
		reduceWrapValidate(),
		reduceAtomic("", nil),
		reduceSepcs(s),
		reduceWrapDone(),
	}

	if resolve {
		reducers = append(reducers, []QueryNodeReducer{
			reduceWrapResolve(s),
			reduceWrapDone(),
			reduceWrapValidate(),
			reduceAtomic("", nil),
			reduceSepcs(s),
			reduceWrapDone(),
		}...)
	}

	reducers = append(reducers, reduceSnapshot())

	queryName := "Snapshot"
	if resolve {
		queryName = "Resolve"
	}
	q, err := s.NewQuery(queryName, "Environment", varDefs,
		reducers,
	)
	if err != nil {
		return err
	}

	text, err := q.Print()
	if err != nil {
		return err
	}

	_, err = query.WriteString(text)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) defineEnvSpecDefsQuery(query io.StringWriter) error {
	varDefs := []*ast.VariableDefinition{
		ast.NewVariableDefinition(&ast.VariableDefinition{
			Variable: ast.NewVariable(&ast.Variable{
				Name: ast.NewName(&ast.Name{
					Value: "definitions",
				}),
			}),
			Type: ast.NewNamed(&ast.Named{
				Name: ast.NewName(&ast.Name{
					Value: "[EnvSpecInputType]!",
				}),
			}),
		}),
	}
	q, err := s.NewQuery(
		"EnvSpecsDef",
		"EnvSpecs",
		varDefs,
		[]QueryNodeReducer{
			func(opSets []*OperationSet, opDef *ast.OperationDefinition, selSet *ast.SelectionSet) (*ast.SelectionSet, error) {
				nextSelSet := ast.NewSelectionSet(&ast.SelectionSet{
					Selections: []ast.Selection{
						ast.NewField(&ast.Field{
							Name: ast.NewName(&ast.Name{
								Value: "name",
							}),
						}),
						ast.NewField(&ast.Field{
							Name: ast.NewName(&ast.Name{
								Value: "breaker",
							}),
						}),
						ast.NewField(&ast.Field{
							Name: ast.NewName(&ast.Name{
								Value: "origin",
							}),
						}),
						ast.NewField(&ast.Field{
							Name: ast.NewName(&ast.Name{
								Value: "atomics",
							}),
							SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
								Selections: []ast.Selection{
									ast.NewField(&ast.Field{
										Name: ast.NewName(&ast.Name{
											Value: "key",
										}),
									}),
									ast.NewField(&ast.Field{
										Name: ast.NewName(&ast.Name{
											Value: "atomic",
										}),
									}),
									ast.NewField(&ast.Field{
										Name: ast.NewName(&ast.Name{
											Value: "rules",
										}),
									}),
									ast.NewField(&ast.Field{
										Name: ast.NewName(&ast.Name{
											Value: "required",
										}),
									}),
								},
							}),
						}),
					},
				})
				selSet.Selections = append(selSet.Selections, ast.NewField(&ast.Field{
					Name: ast.NewName(&ast.Name{
						Value: "load",
					}),
					Arguments: []*ast.Argument{
						ast.NewArgument(&ast.Argument{
							Name: ast.NewName(&ast.Name{
								Value: "definitions",
							}),
							Value: ast.NewVariable(&ast.Variable{
								Name: ast.NewName(&ast.Name{
									Value: "definitions",
								}),
							}),
						}),
					},
					SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
						Selections: []ast.Selection{
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "definitions",
								}),
								SelectionSet: nextSelSet,
							}),
						},
					}),
				}))
				return nil, nil
			},
		},
	)
	if err != nil {
		return err
	}

	text, err := q.Print()
	if err != nil {
		return err
	}

	_, err = query.WriteString(text)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) sensitiveKeysQuery(query, vars io.StringWriter) error {
	varDefs := []*ast.VariableDefinition{
		ast.NewVariableDefinition(&ast.VariableDefinition{
			Variable: ast.NewVariable(&ast.Variable{
				Name: ast.NewName(&ast.Name{
					Value: "insecure",
				}),
			}),
			Type: ast.NewNamed(&ast.Named{
				Name: ast.NewName(&ast.Name{
					Value: "Boolean",
				}),
			}),
			DefaultValue: ast.NewBooleanValue(&ast.BooleanValue{
				Value: false,
			}),
		}),
	}

	q, err := s.NewQuery("Sensitive", "Environment", varDefs,
		[]QueryNodeReducer{
			reconcileAsymmetry(s),
			reduceSetOperations(vars),
			reduceWrapValidate(),
			reduceAtomic("", nil),
			reduceSepcs(s),
			reduceWrapDone(),
			reduceSensitive(),
		},
	)
	if err != nil {
		return err
	}

	text, err := q.Print()
	if err != nil {
		return err
	}

	_, err = query.WriteString(text)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) getterQuery(query, vars io.StringWriter) error {
	varDefs := []*ast.VariableDefinition{
		ast.NewVariableDefinition(&ast.VariableDefinition{
			Variable: ast.NewVariable(&ast.Variable{
				Name: ast.NewName(&ast.Name{
					Value: "key",
				}),
			}),
			Type: ast.NewNamed(&ast.Named{
				Name: ast.NewName(&ast.Name{
					Value: "String",
				}),
			}),
			DefaultValue: ast.NewStringValue(&ast.StringValue{
				Value: "",
			}),
		}),
		ast.NewVariableDefinition(&ast.VariableDefinition{
			Variable: ast.NewVariable(&ast.Variable{
				Name: ast.NewName(&ast.Name{
					Value: "insecure",
				}),
			}),
			Type: ast.NewNamed(&ast.Named{
				Name: ast.NewName(&ast.Name{
					Value: "Boolean",
				}),
			}),
			DefaultValue: ast.NewBooleanValue(&ast.BooleanValue{
				Value: false,
			}),
		}),
	}

	loaded, updated, deleted := 0, 0, 0
	for _, opSet := range s.opSets {
		if len(opSet.specs) == 0 && len(opSet.values) == 0 {
			continue
		}
		switch opSet.operation.kind {
		case LoadSetOperation:
			loaded++
		case UpdateSetOperation:
			updated++
		case DeleteSetOperation:
			deleted++
		}

	}
	s.logger.Debug("getter opSets breakdown", zap.Int("loaded", loaded), zap.Int("updated", updated), zap.Int("deleted", deleted), zap.Int("total", len(s.opSets)))

	q, err := s.NewQuery("Get", "Environment", varDefs,
		[]QueryNodeReducer{
			reconcileAsymmetry(s),
			reduceSetOperations(vars),
			reduceWrapValidate(),
			reduceAtomic("", nil),
			reduceSepcs(s),
			reduceWrapDone(),
			reduceGetter(),
		},
	)
	if err != nil {
		return err
	}

	text, err := q.Print()
	if err != nil {
		return err
	}

	_, err = query.WriteString(text)
	if err != nil {
		return err
	}

	return nil
}

func reduceWrapResolve(store *Store) QueryNodeReducer {
	exprVal := `key | lower()`
	projectVal := "dev"

	// todo(sebastian): we should traverse the path and gen the query
	if store.resolvePath != nil {
		if t, err := extractDataKey(store.resolvePath, "transform"); err == nil {
			if expr, err := extractDataKey(t, "expr"); err == nil {
				exprVal = expr.(string)
			}
			if gcp, err := extractDataKey(t, "gcp"); err == nil {
				if auth, err := extractDataKey(gcp, "auth"); err == nil {
					if v, ok := auth.(string); !ok || v != "ADC" {
						return nil
					}
				}
				if project, err := extractDataKey(gcp, "project"); err == nil {
					projectVal = project.(string)
				}
			}
		}
	}
	return func(opSets []*OperationSet, opDef *ast.OperationDefinition, selSet *ast.SelectionSet) (*ast.SelectionSet, error) {
		resolveSelSet := ast.NewSelectionSet(&ast.SelectionSet{
			Selections: []ast.Selection{
				ast.NewField(&ast.Field{
					Name: ast.NewName(&ast.Name{
						Value: "mapping",
					}),
				}),
			},
		})
		selSet.Selections = append(selSet.Selections,
			ast.NewField(&ast.Field{
				Name: ast.NewName(&ast.Name{
					Value: "resolve",
				}),
				SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
					Selections: []ast.Selection{
						ast.NewField(&ast.Field{
							Name: ast.NewName(&ast.Name{
								Value: "GcpProvider",
							}),
							Arguments: []*ast.Argument{
								ast.NewArgument(&ast.Argument{
									Name: ast.NewName(&ast.Name{
										Value: "api",
									}),
									Value: ast.NewStringValue(&ast.StringValue{
										Value: "secretmanager.apiv1",
									}),
								}),
								ast.NewArgument(&ast.Argument{
									Name: ast.NewName(&ast.Name{
										Value: "project",
									}),
									Value: ast.NewStringValue(&ast.StringValue{
										Value: projectVal,
									}),
								}),
							},
							SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
								Selections: []ast.Selection{
									ast.NewField(&ast.Field{
										Name: ast.NewName(&ast.Name{
											Value: "transform",
										}),
										Arguments: []*ast.Argument{
											ast.NewArgument(&ast.Argument{
												Name: ast.NewName(&ast.Name{
													Value: "expr",
												}),
												Value: ast.NewStringValue(&ast.StringValue{
													Value: exprVal,
												}),
											}),
										},
										SelectionSet: resolveSelSet,
									}),
								},
							}),
						}),
					},
				}),
			}))
		return resolveSelSet, nil
	}
}

func reduceWrapValidate() QueryNodeReducer {
	return func(opSets []*OperationSet, opDef *ast.OperationDefinition, selSet *ast.SelectionSet) (*ast.SelectionSet, error) {
		validateSelSet := ast.NewSelectionSet(&ast.SelectionSet{})
		selSet.Selections = append(selSet.Selections, ast.NewField(&ast.Field{
			Name: ast.NewName(&ast.Name{
				Value: "validate",
			}),
			SelectionSet: validateSelSet,
		}))
		return validateSelSet, nil
	}
}

func reduceWrapDone() QueryNodeReducer {
	return func(opSets []*OperationSet, opDef *ast.OperationDefinition, selSet *ast.SelectionSet) (*ast.SelectionSet, error) {
		doneSelSet := ast.NewSelectionSet(&ast.SelectionSet{})
		selSet.Selections = append(selSet.Selections, ast.NewField(&ast.Field{
			Name: ast.NewName(&ast.Name{
				Value: "done",
			}),
			SelectionSet: doneSelSet,
		}))
		return doneSelSet, nil
	}
}

func reduceSetOperations(vars io.StringWriter) QueryNodeReducer {
	return func(opSets []*OperationSet, opDef *ast.OperationDefinition, selSet *ast.SelectionSet) (*ast.SelectionSet, error) {
		opSetData := make(map[string]SetVarItems, len(opSets))

		for i, opSet := range opSets {
			if len(opSet.values) == 0 && len(opSet.specs) == 0 {
				continue
			}

			opName := ""
			switch opSet.operation.kind {
			case LoadSetOperation:
				opName = "load"
			case UpdateSetOperation:
				opName = "update"
			case ReconcileSetOperation:
				opName = "reconcile"
			case DeleteSetOperation:
				opName = "delete"
			default:
				continue
			}
			nvars := fmt.Sprintf("%s_%d", opName, i)

			for _, v := range opSet.values {
				opSetData[nvars] = append(opSetData[nvars], &SetVarItem{
					Var:   v.Var,
					Value: v.Value,
				})
			}

			for _, s := range opSet.specs {
				opSetData[nvars] = append(opSetData[nvars], &SetVarItem{
					Var:  s.Var,
					Spec: s.Spec,
				})
			}

			opDef.VariableDefinitions = append(opDef.VariableDefinitions, ast.NewVariableDefinition(&ast.VariableDefinition{
				Variable: ast.NewVariable(&ast.Variable{
					Name: ast.NewName(&ast.Name{
						Value: nvars,
					}),
				}),
				Type: ast.NewNamed(&ast.Named{
					Name: ast.NewName(&ast.Name{
						Value: "[VariableInput]!",
					}),
				}),
			}))

			nextSelSet := ast.NewSelectionSet(&ast.SelectionSet{})
			// nextSelSet.Selections = append(nextSelSet.Selections, ast.NewField(&ast.Field{
			// 	Name: ast.NewName(&ast.Name{
			// 		Value: "location",
			// 	}),
			// }))
			selSet.Selections = append(selSet.Selections, ast.NewField(&ast.Field{
				Name: ast.NewName(&ast.Name{
					Value: opName,
				}),
				Arguments: []*ast.Argument{
					ast.NewArgument(&ast.Argument{
						Name: ast.NewName(&ast.Name{
							Value: "vars",
						}),
						Value: ast.NewVariable(&ast.Variable{
							Name: ast.NewName(&ast.Name{
								Value: nvars,
							}),
						}),
					}),
					// ast.NewArgument(&ast.Argument{
					// 	Name: ast.NewName(&ast.Name{
					// 		Value: "location",
					// 	}),
					// 	Value: ast.NewStringValue(&ast.StringValue{
					// 		Value: opSet.operation.location,
					// 	}),
					// }),
					ast.NewArgument(&ast.Argument{
						Name: ast.NewName(&ast.Name{
							Value: "hasSpecs",
						}),
						Value: ast.NewBooleanValue(&ast.BooleanValue{
							Value: opSet.hasSpecs,
						}),
					}),
				},
				Directives:   []*ast.Directive{},
				SelectionSet: nextSelSet,
			}))
			selSet = nextSelSet
		}

		opSetJSON, err := json.MarshalIndent(opSetData, "", " ")
		if err != nil {
			return nil, err
		}
		_, err = vars.WriteString(string(opSetJSON))
		if err != nil {
			return nil, err
		}

		return selSet, nil
	}
}

func reduceSensitive() QueryNodeReducer {
	return func(opSets []*OperationSet, opDef *ast.OperationDefinition, selSet *ast.SelectionSet) (*ast.SelectionSet, error) {
		nextSelSet := ast.NewSelectionSet(&ast.SelectionSet{
			Selections: []ast.Selection{
				ast.NewField(&ast.Field{
					Name: ast.NewName(&ast.Name{
						Value: "var",
					}),
					SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
						Selections: []ast.Selection{
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "key",
								}),
							}),
						},
					}),
				}),
				// ast.NewField(&ast.Field{
				// 	Name: ast.NewName(&ast.Name{
				// 		Value: "value",
				// 	}),
				// 	SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
				// 		Selections: []ast.Selection{
				// 			ast.NewField(&ast.Field{
				// 				Name: ast.NewName(&ast.Name{
				// 					Value: "original",
				// 				}),
				// 			}),
				// 			ast.NewField(&ast.Field{
				// 				Name: ast.NewName(&ast.Name{
				// 					Value: "resolved",
				// 				}),
				// 			}),
				// 			ast.NewField(&ast.Field{
				// 				Name: ast.NewName(&ast.Name{
				// 					Value: "status",
				// 				}),
				// 			}),
				// 		},
				// 	}),
				// }),
				ast.NewField(&ast.Field{
					Name: ast.NewName(&ast.Name{
						Value: "spec",
					}),
					SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
						Selections: []ast.Selection{
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "name",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "description",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "required",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "checked",
								}),
							}),
						},
					}),
				}),
				ast.NewField(&ast.Field{
					Name: ast.NewName(&ast.Name{
						Value: "errors",
					}),
					SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
						Selections: []ast.Selection{
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "code",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "message",
								}),
							}),
						},
					}),
				}),
			},
		})

		selSet.Selections = append(selSet.Selections,
			ast.NewField(&ast.Field{
				Name: ast.NewName(&ast.Name{
					Value: "render",
				}),
				SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
					Selections: []ast.Selection{
						ast.NewField(&ast.Field{
							Name: ast.NewName(&ast.Name{
								Value: "sensitiveKeys",
							}),
							SelectionSet: nextSelSet,
						}),
					},
				}),
			}),
		)

		return nextSelSet, nil
	}
}

func reduceGetter() QueryNodeReducer {
	return func(opSets []*OperationSet, opDef *ast.OperationDefinition, selSet *ast.SelectionSet) (*ast.SelectionSet, error) {
		nextSelSet := ast.NewSelectionSet(&ast.SelectionSet{
			Selections: []ast.Selection{
				ast.NewField(&ast.Field{
					Name: ast.NewName(&ast.Name{
						Value: "var",
					}),
					SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
						Selections: []ast.Selection{
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "key",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "origin",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "created",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "updated",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "operation",
								}),
								SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
									Selections: []ast.Selection{
										ast.NewField(&ast.Field{
											Name: ast.NewName(&ast.Name{
												Value: "source",
											}),
										}),
									},
								}),
							}),
						},
					}),
				}),
				ast.NewField(&ast.Field{
					Name: ast.NewName(&ast.Name{
						Value: "value",
					}),
					SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
						Selections: []ast.Selection{
							// ast.NewField(&ast.Field{
							// 	Name: ast.NewName(&ast.Name{
							// 		Value: "type",
							// 	}),
							// }),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "original",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "resolved",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "status",
								}),
							}),
						},
					}),
				}),
				ast.NewField(&ast.Field{
					Name: ast.NewName(&ast.Name{
						Value: "spec",
					}),
					SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
						Selections: []ast.Selection{
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "name",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "description",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "required",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "checked",
								}),
							}),
						},
					}),
				}),
				// ast.NewField(&ast.Field{
				// 	Name: ast.NewName(&ast.Name{
				// 		Value: "errors",
				// 	}),
				// 	SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
				// 		Selections: []ast.Selection{
				// 			ast.NewField(&ast.Field{
				// 				Name: ast.NewName(&ast.Name{
				// 					Value: "code",
				// 				}),
				// 			}),
				// 			ast.NewField(&ast.Field{
				// 				Name: ast.NewName(&ast.Name{
				// 					Value: "message",
				// 				}),
				// 			}),
				// 		},
				// 	}),
				// }),
			},
		})

		selSet.Selections = append(selSet.Selections,
			ast.NewField(&ast.Field{
				Name: ast.NewName(&ast.Name{
					Value: "render",
				}),
				SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
					Selections: []ast.Selection{
						ast.NewField(&ast.Field{
							Name: ast.NewName(&ast.Name{
								Value: "get",
							}),
							Arguments: []*ast.Argument{
								ast.NewArgument(&ast.Argument{
									Name: ast.NewName(&ast.Name{
										Value: "key",
									}),
									Value: ast.NewVariable(&ast.Variable{
										Name: ast.NewName(&ast.Name{
											Value: "key",
										}),
									}),
								}),
							},
							SelectionSet: nextSelSet,
						}),
					},
				}),
			}),
		)

		return nextSelSet, nil
	}
}

func reduceSnapshot() QueryNodeReducer {
	return func(opSets []*OperationSet, opDef *ast.OperationDefinition, selSet *ast.SelectionSet) (*ast.SelectionSet, error) {
		nextSelSet := ast.NewSelectionSet(&ast.SelectionSet{
			Selections: []ast.Selection{
				ast.NewField(&ast.Field{
					Name: ast.NewName(&ast.Name{
						Value: "var",
					}),
					SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
						Selections: []ast.Selection{
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "key",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "origin",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "created",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "updated",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "operation",
								}),
								SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
									Selections: []ast.Selection{
										ast.NewField(&ast.Field{
											Name: ast.NewName(&ast.Name{
												Value: "source",
											}),
										}),
									},
								}),
							}),
						},
					}),
				}),
				ast.NewField(&ast.Field{
					Name: ast.NewName(&ast.Name{
						Value: "value",
					}),
					SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
						Selections: []ast.Selection{
							// ast.NewField(&ast.Field{
							// 	Name: ast.NewName(&ast.Name{
							// 		Value: "type",
							// 	}),
							// }),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "original",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "resolved",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "status",
								}),
							}),
						},
					}),
				}),
				ast.NewField(&ast.Field{
					Name: ast.NewName(&ast.Name{
						Value: "spec",
					}),
					SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
						Selections: []ast.Selection{
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "name",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "description",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "required",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "checked",
								}),
							}),
						},
					}),
				}),
				ast.NewField(&ast.Field{
					Name: ast.NewName(&ast.Name{
						Value: "errors",
					}),
					SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
						Selections: []ast.Selection{
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "code",
								}),
							}),
							ast.NewField(&ast.Field{
								Name: ast.NewName(&ast.Name{
									Value: "message",
								}),
							}),
						},
					}),
				}),
			},
		})

		selSet.Selections = append(selSet.Selections,
			ast.NewField(&ast.Field{
				Name: ast.NewName(&ast.Name{
					Value: "render",
				}),
				SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
					Selections: []ast.Selection{
						ast.NewField(&ast.Field{
							Name: ast.NewName(&ast.Name{
								Value: "snapshot",
							}),
							Arguments: []*ast.Argument{
								ast.NewArgument(&ast.Argument{
									Name: ast.NewName(&ast.Name{
										Value: "insecure",
									}),
									Value: ast.NewVariable(&ast.Variable{
										Name: ast.NewName(&ast.Name{
											Value: "insecure",
										}),
									}),
								}),
							},
							SelectionSet: nextSelSet,
						}),
					},
				}),
			}),
		)

		return nextSelSet, nil
	}
}

func reconcileAsymmetry(store *Store) QueryNodeReducer {
	return func(opSets []*OperationSet, opDef *ast.OperationDefinition, selSet *ast.SelectionSet) (*ast.SelectionSet, error) {
		allSpecs := make(map[string]bool)
		for _, opSet := range store.opSets {
			for k := range opSet.specs {
				allSpecs[k] = true
			}
		}
		allVals := make(map[string]bool)
		for _, opSet := range store.opSets {
			for k := range opSet.values {
				allVals[k] = true
			}
		}

		deltaOpSet, err := NewOperationSet(WithOperation(ReconcileSetOperation))
		if err != nil {
			return nil, err
		}

		for _, opSet := range store.opSets {
			for k := range opSet.values {
				if _, exists := allSpecs[k]; exists {
					continue
				}
				created := time.Now()
				spec := &SetVarSpec{
					Var: &SetVar{
						Key:     k,
						Created: &created,
					},
					Spec: &varSpec{
						Name:     AtomicNameDefault,
						Required: false,
						Checked:  false,
					},
				}
				deltaOpSet.specs[k] = spec
			}
			for k := range opSet.specs {
				if _, exists := allVals[k]; exists {
					continue
				}
				created := time.Now()
				val := &SetVarValue{
					Var: &SetVar{
						Key:     k,
						Created: &created,
					},
					Value: &varValue{
						Status: "UNRESOLVED",
						Operation: &setVarOperation{
							Kind: ReconcileSetOperation,
						},
					},
				}
				deltaOpSet.values[k] = val
			}
		}

		if len(deltaOpSet.specs) > 0 || len(deltaOpSet.values) > 0 {
			deltaOpSet.hasSpecs = len(deltaOpSet.specs) > 0
			store.opSets = append(store.opSets, deltaOpSet)
		}

		return selSet, nil
	}
}

func reduceAtomic(ns string, parent *SpecDef) QueryNodeReducer {
	return func(opSets []*OperationSet, opDef *ast.OperationDefinition, selSet *ast.SelectionSet) (*ast.SelectionSet, error) {
		var specKeys []string
		varSpecs := make(map[string]*SetVarItem)
		for _, opSet := range opSets {
			if len(opSet.specs) == 0 {
				continue
			}
			for _, s := range opSet.specs {
				isTransient := opSet.operation.kind == TransientSetOperation
				if isTransient && parent != nil {
					for k, v := range parent.Atomics {
						if fmt.Sprintf("%s_%s", ns, k) != s.Var.Key {
							continue
						}
						s.Spec = v
					}
				}

				if _, ok := AtomicTypes[s.Spec.Name]; !ok {
					// return nil, fmt.Errorf("unknown spec type: %s on %s", s.Spec.Name, s.Var.Key)
					continue
				}

				varSpecs[s.Var.Key] = &SetVarItem{
					Var:  s.Var,
					Spec: s.Spec,
				}
				specKeys = append(specKeys, s.Spec.Name)
			}
		}

		reduceVarSpecs := func(varSpecs map[string]*SetVarItem, spec string, prevSelSet *ast.SelectionSet) *ast.SelectionSet {
			var keys []string
			for _, v := range varSpecs {
				if v.Spec.Name != spec {
					continue
				}
				keys = append(keys, v.Var.Key)
			}
			if len(keys) == 0 {
				return prevSelSet
			}

			nextSelSet := ast.NewSelectionSet(&ast.SelectionSet{
				Selections: []ast.Selection{
					ast.NewField(&ast.Field{
						Name: ast.NewName(&ast.Name{
							Value: "name",
						}),
					}),
					ast.NewField(&ast.Field{
						Name: ast.NewName(&ast.Name{
							Value: "sensitive",
						}),
					}),
					ast.NewField(&ast.Field{
						Name: ast.NewName(&ast.Name{
							Value: "mask",
						}),
					}),
					// ast.NewField(&ast.Field{
					// 	Name: ast.NewName(&ast.Name{
					// 		Value: "errors",
					// 	}),
					// }),
				},
			})

			valuekeys := ast.NewListValue(&ast.ListValue{})
			for _, name := range keys {
				valuekeys.Values = append(valuekeys.Values, ast.NewStringValue(&ast.StringValue{
					Value: name,
				}))
			}

			prevSelSet.Selections = append(prevSelSet.Selections,
				ast.NewField(&ast.Field{
					Name: ast.NewName(&ast.Name{
						Value: spec,
					}),
					Arguments: []*ast.Argument{
						ast.NewArgument(&ast.Argument{
							Name: ast.NewName(&ast.Name{
								Value: "insecure",
							}),
							Value: ast.NewVariable(&ast.Variable{
								Name: ast.NewName(&ast.Name{
									Value: "insecure",
								}),
							}),
						}),
						ast.NewArgument(&ast.Argument{
							Name: ast.NewName(&ast.Name{
								Value: "keys",
							}),
							Value: valuekeys,
						}),
					},
					SelectionSet: nextSelSet,
				}))

			return nextSelSet
		}

		nextSelSet := selSet

		// todo: poor Sebastian's deduplication
		slices.Sort(specKeys)
		prev := ""
		for _, specKey := range specKeys {
			if prev == specKey {
				continue
			}
			prev = specKey
			nextSelSet = reduceVarSpecs(varSpecs, specKey, nextSelSet)
		}

		return nextSelSet, nil
	}
}

func reduceSepcs(store *Store) QueryNodeReducer {
	return func(opSets []*OperationSet, opDef *ast.OperationDefinition, selSet *ast.SelectionSet) (*ast.SelectionSet, error) {
		var varKeys []string
		varSpecs := make(map[string]*SetVarItem)
		for _, opSet := range opSets {
			if len(opSet.specs) == 0 {
				continue
			}
			for _, s := range opSet.specs {
				if _, ok := AtomicTypes[s.Spec.Name]; ok {
					continue
				}
				varSpecs[s.Var.Key] = &SetVarItem{
					Var:  s.Var,
					Spec: s.Spec,
				}
				varKeys = append(varKeys, s.Var.Key)
			}
		}

		reduceNamespacedSpec := func(varSpecs map[string]*SetVarItem, ns string, prevSelSet *ast.SelectionSet) *ast.SelectionSet {
			spec := ""
			var keys []string
			var items SetVarItems
			for _, v := range varSpecs {
				spec = v.Spec.Name
				keys = append(keys, v.Var.Key)
				items = append(items, v)
			}
			if len(keys) == 0 {
				return prevSelSet
			}

			nextSelSet := ast.NewSelectionSet(&ast.SelectionSet{
				Selections: []ast.Selection{
					ast.NewField(&ast.Field{
						Name: ast.NewName(&ast.Name{
							Value: "name",
						}),
					}),
					// ast.NewField(&ast.Field{
					// 	Name: ast.NewName(&ast.Name{
					// 		Value: "sensitive",
					// 	}),
					// }),
					// ast.NewField(&ast.Field{
					// 	Name: ast.NewName(&ast.Name{
					// 		Value: "mask",
					// 	}),
					// }),
					// ast.NewField(&ast.Field{
					// 	Name: ast.NewName(&ast.Name{
					// 		Value: "errors",
					// 	}),
					// }),
				},
			})

			valuekeys := ast.NewListValue(&ast.ListValue{})
			for _, name := range keys {
				valuekeys.Values = append(valuekeys.Values, ast.NewStringValue(&ast.StringValue{
					Value: name,
				}))
			}

			prevSelSet.Selections = append(prevSelSet.Selections,
				ast.NewField(&ast.Field{
					Name: ast.NewName(&ast.Name{
						Value: SpecTypeKey,
					}),
					Arguments: []*ast.Argument{
						ast.NewArgument(&ast.Argument{
							Name: ast.NewName(&ast.Name{
								Value: "name",
							}),
							Value: ast.NewStringValue(&ast.StringValue{
								Value: spec,
							}),
						}),
						ast.NewArgument(&ast.Argument{
							Name: ast.NewName(&ast.Name{
								Value: "namespace",
							}),
							Value: ast.NewStringValue(&ast.StringValue{
								Value: ns,
							}),
						}),
						ast.NewArgument(&ast.Argument{
							Name: ast.NewName(&ast.Name{
								Value: "keys",
							}),
							Value: valuekeys,
						}),
					},
					SelectionSet: nextSelSet,
				}))

			specDef := store.specDefs[spec]

			transientOpSet, _ := NewOperationSet(WithOperation(TransientSetOperation), WithItems(items))
			atomicSelSet, err := reduceAtomic(ns, specDef)([]*OperationSet{transientOpSet}, opDef, nextSelSet)
			// todo: handle error
			if err == nil {
				return atomicSelSet
			}

			return nextSelSet
		}

		nextSelSet := selSet

		slices.Sort(varKeys)
		namespaced := make(map[string]map[string]*SetVarItem)
		// todo: generalize
		breaker := ""
		for _, specKey := range varKeys {
			ns := ""

			item := varSpecs[specKey]
			specDef, ok := store.specDefs[item.Spec.Name]
			if !ok {
				return nil, fmt.Errorf("unknown spec type: %s on %s", item.Spec.Name, item.Var.Key)
			}

			breaker = specDef.Breaker
			parts := strings.Split(specKey, "_")
			for i, part := range parts {
				if part == breaker {
					ns = strings.Join(parts[:i+1], "_")
					break
				}
			}

			if namespaced[ns] == nil {
				namespaced[ns] = make(map[string]*SetVarItem)
			}

			// if item.Spec.Name != prevNs && prevNs != "" {
			// 	return nil, fmt.Errorf("spec type mismatch in namespace %q: %s != %s", ns, item.Spec.Name, prevNs)
			// }
			// prevNs = ns

			namespaced[ns][specKey] = item
		}

		for ns, vars := range namespaced {
			nextSelSet = reduceNamespacedSpec(vars, ns, nextSelSet)
		}

		return nextSelSet, nil
	}
}

type Query struct {
	doc *ast.Document
}

func (s *Store) NewQuery(queryName, rootSelelection string, varDefs []*ast.VariableDefinition, reducers []QueryNodeReducer) (*Query, error) {
	selSet := ast.NewSelectionSet(&ast.SelectionSet{})
	opDef := ast.NewOperationDefinition(&ast.OperationDefinition{
		Operation: "query",
		Name: ast.NewName(&ast.Name{
			Value: fmt.Sprintf("Owl%s", queryName),
		}),
		Directives: []*ast.Directive{},
		SelectionSet: ast.NewSelectionSet(&ast.SelectionSet{
			Selections: []ast.Selection{
				ast.NewField(&ast.Field{
					Name: ast.NewName(&ast.Name{
						Value: rootSelelection,
					}),
					Arguments:    []*ast.Argument{},
					Directives:   []*ast.Directive{},
					SelectionSet: selSet,
				}),
			},
		}),
		VariableDefinitions: varDefs,
	})

	var err error
	for _, reducer := range reducers {
		if reducer == nil {
			continue
		}
		if selSet, err = reducer(s.opSets, opDef, selSet); err != nil {
			return nil, err
		}
	}

	doc := ast.NewDocument(&ast.Document{Definitions: []ast.Node{opDef}})

	return &Query{doc: doc}, nil
}

func (q *Query) Print() (string, error) {
	res := printer.Print(q.doc)
	text, ok := res.(string)
	if !ok {
		return "", errors.New("ast printer returned unknown type")
	}

	return text, nil
}
